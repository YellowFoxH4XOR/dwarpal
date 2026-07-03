// Package config loads and validates .dwarpal.yml.
//
// Two rules shape this package (design decision D6):
//  1. Compiled-in defaults always apply; a config file only overlays them, so a
//     partial file (setting one budget) leaves the rest at their defaults.
//  2. Validation is strict and fails closed. An unknown key or an out-of-domain
//     value exits the process (code 2) naming the offender, because a security
//     gate whose misconfiguration is silently ignored is worse than no gate.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Mode controls whether findings block. See PRD §5.3.
type Mode string

const (
	ModeEnforce  Mode = "enforce"   // error findings block (exit 1)
	ModeWarn     Mode = "warn"      // findings reported, exit 0
	ModeCIStrict Mode = "ci_strict" // enforce + bypasses rejected (server-side; M1+)
)

// Config is the whole validated configuration.
type Config struct {
	Version          int             `koanf:"version"`
	Mode             Mode            `koanf:"mode"`
	StopOnFirstBlock bool            `koanf:"stop_on_first_block"`
	Provenance       ProvenanceBlock `koanf:"provenance"`
	Gates            GatesBlock      `koanf:"gates"`
	ArchRules        []ArchRule      `koanf:"architecture_rules"`
	// RuleOverrides reassigns a rule's severity, keyed by "gate/rule_id" (e.g.
	// "ai_patterns/no-sql-concat": "warn"). Authored by hand, by an agent, or by
	// `dwarpal audit --apply` from the measured acted-on rate.
	RuleOverrides map[string]string `koanf:"rule_overrides"`
}

// ArchRule is one user-defined architecture assertion (PRD §5.3). Calls whose
// rendered target matches Matches are FORBIDDEN outside the ForbiddenOutside
// globs. Query is accepted for forward-compat with tree-sitter query syntax
// but ignored by the v1 Go (go/parser) implementation.
type ArchRule struct {
	ID               string   `koanf:"id"`
	Description      string   `koanf:"description"`
	Language         string   `koanf:"language"`
	Query            string   `koanf:"query"`
	Matches          string   `koanf:"matches"`
	ForbiddenOutside []string `koanf:"forbidden_outside"`
	Severity         string   `koanf:"severity"`
}

// ProvenanceBlock configures agent detection and which commits gates apply to.
type ProvenanceBlock struct {
	BranchPrefixes []string `koanf:"branch_prefixes"`
	Trailers       []string `koanf:"trailers"`
	// Heuristics are user-defined regexes matched against branch name and
	// commit message — the PRD's fourth, weakest detection signal.
	Heuristics   []string `koanf:"heuristics"`
	ApplyGatesTo string   `koanf:"apply_gates_to"` // agent-only | all-commits
}

// GatesBlock groups per-gate configuration.
type GatesBlock struct {
	DiffBudget      DiffBudget     `koanf:"diff_budget"`
	BranchPolicy    BranchPolicy   `koanf:"branch_policy"`
	AIPatterns      AIPatterns     `koanf:"ai_patterns"`
	Scope           ScopeBlock     `koanf:"scope"`
	DiffCoverage    DiffCoverage   `koanf:"diff_coverage"`
	IntentCheck     IntentCheck    `koanf:"intent_check"`
	Duplicate       DuplicateBlock `koanf:"duplicate"`
	ConventionDrift DriftBlock     `koanf:"convention_drift"`
	Plugins         []Plugin       `koanf:"plugins"`
}

// DuplicateBlock is the no-duplicate-function rule's config. Opt-in: it builds
// the repo function index, so it is off by default to protect the p95 budget.
type DuplicateBlock struct {
	Enabled   bool    `koanf:"enabled"`
	Threshold float64 `koanf:"threshold"`
}

// DriftBlock is Gate 6's config. Advisory (info) and on by default.
type DriftBlock struct {
	Enabled  bool   `koanf:"enabled"`
	Severity string `koanf:"severity"`
}

// DiffCoverage is Gate 5's configuration. The gate is active only when Artifact
// is set (Dwarpal consumes an existing coverage artifact, never runs tests).
type DiffCoverage struct {
	MinPercent float64 `koanf:"min_percent"`
	Artifact   string  `koanf:"artifact"`
}

// IntentCheck is Gate 7's configuration. Off by default; the API key is read
// from the environment, never stored in config.
type IntentCheck struct {
	Enabled        bool   `koanf:"enabled"`
	Provider       string `koanf:"provider"`
	Endpoint       string `koanf:"endpoint"`
	Model          string `koanf:"model"`
	TimeoutSeconds int    `koanf:"timeout_seconds"`
}

// BranchPolicy is Gate 2's protected-branch configuration.
type BranchPolicy struct {
	Protected []string `koanf:"protected"`
}

// AIPatterns is Gate 3's configuration.
type AIPatterns struct {
	Enabled      bool     `koanf:"enabled"`
	DisableRules []string `koanf:"disable_rules"`
}

// ScopeBlock is Gate 4's configuration.
type ScopeBlock struct {
	RequireTaskManifest bool     `koanf:"require_task_manifest"`
	AllowAlways         []string `koanf:"allow_always"`
}

// Plugin is one Gate 8 exec-plugin definition.
type Plugin struct {
	Name string   `koanf:"name"`
	Exec string   `koanf:"exec"`
	When []string `koanf:"when"`
}

// ApplyGatesTo values.
const (
	ApplyAgentOnly  = "agent-only"
	ApplyAllCommits = "all-commits"
)

// DiffBudget is Gate 1's configuration.
type DiffBudget struct {
	MaxLines    int              `koanf:"max_lines"`
	MaxFiles    int              `koanf:"max_files"`
	MaxNewFiles int              `koanf:"max_new_files"`
	Overrides   []BudgetOverride `koanf:"overrides"`
}

// BudgetOverride relaxes (or tightens) budgets for files matching any of Paths.
type BudgetOverride struct {
	Paths       []string `koanf:"paths"`
	MaxLines    int      `koanf:"max_lines"`
	MaxFiles    int      `koanf:"max_files"`
	MaxNewFiles int      `koanf:"max_new_files"`
}

// Defaults returns the compiled-in configuration (PRD §5.3 defaults).
func Defaults() Config {
	return Config{
		Version: 1,
		Mode:    ModeEnforce,
		Provenance: ProvenanceBlock{
			BranchPrefixes: []string{"agent/", "ai/"},
			Trailers:       []string{"Claude", "GitHub Copilot", "Cursor", "Devin", "Aider"},
			// Default: gate EVERY commit. Quality rules that only apply to
			// some authors invite drift; teams that want humans exempt opt
			// out explicitly with agent-only.
			ApplyGatesTo: ApplyAllCommits,
		},
		Gates: GatesBlock{
			DiffBudget: DiffBudget{
				MaxLines:    500,
				MaxFiles:    20,
				MaxNewFiles: 10,
			},
			BranchPolicy: BranchPolicy{
				Protected: []string{"main", "release/*"},
			},
			AIPatterns: AIPatterns{
				Enabled: true,
			},
			Duplicate: DuplicateBlock{
				Enabled:   false, // opt-in: builds the repo index
				Threshold: 0.85,
			},
			ConventionDrift: DriftBlock{
				Enabled:  true,
				Severity: "info",
			},
		},
	}
}

// Filename is the config file Dwarpal looks for at the repo root.
const Filename = ".dwarpal.yml"

// allowedKeys is the exhaustive set of flattened top-level keys. koanf keeps
// the overrides slice as a single leaf value, so its inner keys are validated
// by struct decoding rather than listed here.
var allowedKeys = map[string]bool{
	"version":                            true,
	"mode":                               true,
	"stop_on_first_block":                true,
	"architecture_rules":                 true,
	"rule_overrides":                     true,
	"provenance.branch_prefixes":         true,
	"provenance.trailers":                true,
	"provenance.apply_gates_to":          true,
	"provenance.heuristics":              true,
	"gates.diff_budget.max_lines":        true,
	"gates.diff_budget.max_files":        true,
	"gates.diff_budget.max_new_files":    true,
	"gates.diff_budget.overrides":        true,
	"gates.branch_policy.protected":      true,
	"gates.ai_patterns.enabled":          true,
	"gates.ai_patterns.disable_rules":    true,
	"gates.scope.require_task_manifest":  true,
	"gates.scope.allow_always":           true,
	"gates.diff_coverage.min_percent":    true,
	"gates.diff_coverage.artifact":       true,
	"gates.intent_check.enabled":         true,
	"gates.intent_check.provider":        true,
	"gates.intent_check.endpoint":        true,
	"gates.intent_check.model":           true,
	"gates.intent_check.timeout_seconds": true,
	"gates.duplicate.enabled":            true,
	"gates.duplicate.threshold":          true,
	"gates.convention_drift.enabled":     true,
	"gates.convention_drift.severity":    true,
	"gates.plugins":                      true,
}

// Load reads root/.dwarpal.yml, overlaying it on the defaults. A missing file
// is not an error — defaults are returned. Unknown keys or invalid values
// return an error the CLI maps to exit 2.
func Load(root string) (Config, error) {
	cfg := Defaults()
	path := filepath.Join(root, Filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	k := koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return cfg, fmt.Errorf("reading %s: %w", Filename, err)
	}

	if err := rejectUnknownKeys(k); err != nil {
		return cfg, err
	}

	// Overlay onto defaults: koanf/mapstructure only sets keys present in the
	// file, so unset fields retain their default values.
	if err := k.Unmarshal("", &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", Filename, err)
	}

	if err := cfg.validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// rejectUnknownKeys fails on any flattened key not in allowedKeys, naming it.
func rejectUnknownKeys(k *koanf.Koanf) error {
	var unknown []string
	for _, key := range k.Keys() {
		// rule_overrides is a map with dynamic "gate/rule_id" keys, so koanf
		// flattens it to rule_overrides.<something>; allow the whole namespace.
		if allowedKeys[key] || strings.HasPrefix(key, "rule_overrides.") {
			continue
		}
		unknown = append(unknown, key)
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown config key(s): %s", strings.Join(unknown, ", "))
	}
	return nil
}

// validate enforces value domains. Fails closed so a typo can't weaken a gate.
func (c Config) validate() error {
	switch c.Mode {
	case ModeEnforce, ModeWarn, ModeCIStrict:
	default:
		return fmt.Errorf("invalid mode %q (want enforce|warn|ci_strict)", c.Mode)
	}
	for i, h := range c.Provenance.Heuristics {
		if _, err := regexp.Compile(h); err != nil {
			return fmt.Errorf("provenance.heuristics[%d]: invalid regexp %q: %v", i, h, err)
		}
	}
	switch c.Provenance.ApplyGatesTo {
	case "", ApplyAgentOnly, ApplyAllCommits:
	default:
		return fmt.Errorf("invalid provenance.apply_gates_to %q (want agent-only|all-commits)", c.Provenance.ApplyGatesTo)
	}
	b := c.Gates.DiffBudget
	if b.MaxLines < 0 || b.MaxFiles < 0 || b.MaxNewFiles < 0 {
		return fmt.Errorf("diff_budget budgets must not be negative")
	}
	for i, o := range b.Overrides {
		if o.MaxLines < 0 || o.MaxFiles < 0 || o.MaxNewFiles < 0 {
			return fmt.Errorf("diff_budget.overrides[%d]: budgets must not be negative", i)
		}
	}
	for key, sev := range c.RuleOverrides {
		switch sev {
		case "error", "warn", "info":
		default:
			return fmt.Errorf("rule_overrides[%q]: invalid severity %q (want error|warn|info)", key, sev)
		}
	}
	return nil
}
