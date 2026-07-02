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
	Version int        `koanf:"version"`
	Mode    Mode       `koanf:"mode"`
	Gates   GatesBlock `koanf:"gates"`
}

// GatesBlock groups per-gate configuration.
type GatesBlock struct {
	DiffBudget DiffBudget `koanf:"diff_budget"`
}

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
		Gates: GatesBlock{
			DiffBudget: DiffBudget{
				MaxLines:    500,
				MaxFiles:    20,
				MaxNewFiles: 10,
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
	"version":                          true,
	"mode":                             true,
	"gates.diff_budget.max_lines":      true,
	"gates.diff_budget.max_files":      true,
	"gates.diff_budget.max_new_files":  true,
	"gates.diff_budget.overrides":      true,
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
		if !allowedKeys[key] {
			unknown = append(unknown, key)
		}
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
	b := c.Gates.DiffBudget
	if b.MaxLines < 0 || b.MaxFiles < 0 || b.MaxNewFiles < 0 {
		return fmt.Errorf("diff_budget budgets must not be negative")
	}
	for i, o := range b.Overrides {
		if o.MaxLines < 0 || o.MaxFiles < 0 || o.MaxNewFiles < 0 {
			return fmt.Errorf("diff_budget.overrides[%d]: budgets must not be negative", i)
		}
	}
	return nil
}
