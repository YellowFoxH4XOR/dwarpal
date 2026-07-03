package main

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/aipatterns"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/archrules"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/branchpolicy"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/diffbudget"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/diffcoverage"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/drift"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/duplicate"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/intent"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/plugin"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/scope"
	"github.com/YellowFoxH4XOR/dwarpal/internal/provenance"
	"github.com/YellowFoxH4XOR/dwarpal/internal/repoindex"
	"github.com/YellowFoxH4XOR/dwarpal/internal/taskmanifest"
)

// llmAPIKeyEnv is where the intent gate reads its provider key — never config.
const llmAPIKeyEnv = "DWARPAL_LLM_API_KEY"

// buildGates assembles the gate pipeline for a run, applying the provenance
// filter. Branch policy always participates (it self-no-ops for human commits);
// the content gates run only when this commit is in scope for gating —
// i.e. all-commits mode, or agent-only mode and the change is agent-authored.
// This is the R2 mitigation: human commits stay untouched by default.
func buildGates(root string, cfg config.Config) ([]engine.Gate, provenance.Provenance, engine.RepoIndex) {
	branch := currentBranch(root)
	prov := provenance.New(cfg.Provenance.BranchPrefixes, cfg.Provenance.Trailers).
		Detect(branch, "") // no commit message at pre-commit time

	applyContent := cfg.Provenance.ApplyGatesTo == config.ApplyAllCommits || prov.IsAgent

	var idx engine.RepoIndex = engine.NoIndex{}

	// Branch policy is always present (only fires for agents on protected branches).
	gates := []engine.Gate{
		branchpolicy.New(cfg.Gates.BranchPolicy.Protected, branch, prov.IsAgent),
	}
	if !applyContent {
		return gates, prov, idx
	}

	gates = append(gates, diffbudget.New(cfg.Gates.DiffBudget))
	if cfg.Gates.AIPatterns.Enabled {
		gates = append(gates, aipatterns.New(cfg.Gates.AIPatterns.DisableRules))
	}
	// Scope reads the declared task manifest when present; absent, it is
	// warn-only unless the config requires a manifest. The manifest's task id
	// doubles as the intent text for Gate 7 (#42).
	var scopePaths []string
	taskIntent := ""
	if m, ok, _ := taskmanifest.Load(root); ok {
		scopePaths = m.Paths
		taskIntent = m.ID
	} else if ref := taskmanifest.TicketFromBranch(branch); ref != "" {
		taskIntent = ref // #31: ticket reference in the branch name as fallback identity
	}
	gates = append(gates, scope.New(scopePaths, cfg.Gates.Scope.AllowAlways, cfg.Gates.Scope.RequireTaskManifest))

	// Gate 5 is active only when a coverage artifact is configured.
	if cov := cfg.Gates.DiffCoverage; cov.Artifact != "" {
		min := cov.MinPercent
		if min == 0 {
			min = 70 // PRD default
		}
		gates = append(gates, diffcoverage.New(cov.Artifact, min, root))
	}

	// Gate 7 (intent) is opt-in and BYO-key; it never blocks on infra failure.
	if ic := cfg.Gates.IntentCheck; ic.Enabled {
		if g := buildIntentGate(ic, taskIntent); g != nil {
			gates = append(gates, g)
		}
	}

	// Gates 3 (duplicate) and 6 (drift) need the repo function index. Build it
	// once, only when at least one is enabled, so the p95 budget is untouched
	// otherwise. (Incremental caching under .dwarpal/cache/ is future work — B1.)
	dup := cfg.Gates.Duplicate
	drf := cfg.Gates.ConventionDrift
	if dup.Enabled || drf.Enabled {
		if built, err := repoindex.Build(root); err == nil {
			idx = built
		}
	}
	if dup.Enabled {
		threshold := dup.Threshold
		if threshold == 0 {
			threshold = 0.85
		}
		gates = append(gates, duplicate.New(root, threshold))
	}
	if drf.Enabled {
		gates = append(gates, drift.New(root, finding.Severity(drf.Severity)))
	}

	// User-defined architecture rules (#47, PRD §5.3): forbidden-call
	// assertions evaluated over go/ast on added lines.
	if len(cfg.ArchRules) > 0 {
		rules := make([]archrules.Rule, len(cfg.ArchRules))
		for i, r := range cfg.ArchRules {
			rules[i] = archrules.Rule{
				ID: r.ID, Description: r.Description, Language: r.Language,
				Matches: r.Matches, ForbiddenOutside: r.ForbiddenOutside, Severity: r.Severity,
			}
		}
		gates = append(gates, archrules.New(root, rules))
	}

	for _, p := range cfg.Gates.Plugins {
		gates = append(gates, plugin.New(p.Name, p.Exec, p.When, root))
	}
	return gates, prov, idx
}

// buildIntentGate constructs the LLM intent gate from config + the env-held API
// key. Returns nil (gate omitted) when no key is available, so a misconfigured
// intent gate never silently half-works.
func buildIntentGate(ic config.IntentCheck, taskIntent string) engine.Gate {
	key := os.Getenv(llmAPIKeyEnv)
	if key == "" {
		return nil
	}
	timeout := time.Duration(ic.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // PRD default
	}
	var provider intent.Provider
	if ic.Provider == "anthropic" {
		provider = intent.NewAnthropicProvider(ic.Model, key)
	} else {
		provider = intent.NewOpenAIProvider(ic.Endpoint, ic.Model, key)
	}
	return intent.New(provider, taskIntent, timeout)
}

// currentBranch returns the current branch name, or "" if it cannot be
// determined (detached HEAD). symbolic-ref works even on an unborn branch (a
// fresh repo with no commits), where rev-parse --abbrev-ref HEAD errors.
func currentBranch(root string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
