package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/aipatterns"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/branchpolicy"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/diffbudget"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/scope"
	"github.com/YellowFoxH4XOR/dwarpal/internal/provenance"
	"github.com/YellowFoxH4XOR/dwarpal/internal/taskmanifest"
)

// overrideEnv lists rule IDs (comma-separated) a human has approved skipping
// for this run — the staged-mode counterpart of the Dwarpal-Override trailer
// (at pre-commit time no commit message exists to carry a trailer).
const overrideEnv = "DWARPAL_OVERRIDE"

// overrideTrailer is the commit-message trailer that approves skipping a rule
// for the commits that carry it. Only meaningful in --range mode.
const overrideTrailer = "Dwarpal-Override:"

// collectOverrides gathers approved rule overrides from the env var and, when
// a range is being checked, from Dwarpal-Override trailers in that range's
// commit messages. In ci_strict mode neither is honored — the whole point of
// ci_strict is that a local escape hatch carries no authority.
func collectOverrides(root, rangeArg string, mode config.Mode) []string {
	if mode == config.ModeCIStrict {
		return nil
	}
	var out []string
	for _, id := range strings.Split(os.Getenv(overrideEnv), ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	if rangeArg != "" {
		cmd := exec.Command("git", "log", "--format=%B", rangeArg)
		cmd.Dir = root
		if msgs, err := cmd.Output(); err == nil {
			for _, line := range strings.Split(string(msgs), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, overrideTrailer) {
					if id := strings.TrimSpace(strings.TrimPrefix(line, overrideTrailer)); id != "" {
						out = append(out, id)
					}
				}
			}
		}
	}
	return out
}

// severityOverrides converts the config's rule_overrides (map of "gate/rule_id"
// → severity string) into the typed map the engine applies. Invalid severities
// are already rejected by config.validate, so this is a pure conversion.
func severityOverrides(cfg config.Config) map[string]finding.Severity {
	if len(cfg.RuleOverrides) == 0 {
		return nil
	}
	out := make(map[string]finding.Severity, len(cfg.RuleOverrides))
	for k, v := range cfg.RuleOverrides {
		out[k] = finding.Severity(v)
	}
	return out
}

// buildGates assembles the gate pipeline for a run, applying the provenance
// filter. Branch policy always participates (it self-no-ops for human commits);
// the content gates run only when this commit is in scope for gating — all-
// commits mode, or agent-only mode and the change is agent-authored. This is
// the R2 mitigation: human commits stay untouched by default.
func buildGates(root string, cfg config.Config, overrides []string) ([]engine.Gate, provenance.Provenance) {
	branch := currentBranch(root)
	prov := provenance.New(cfg.Provenance.BranchPrefixes, cfg.Provenance.Trailers).
		WithHeuristics(cfg.Provenance.Heuristics).
		Detect(branch, "") // no commit message at pre-commit time

	applyContent := cfg.Provenance.ApplyGatesTo == config.ApplyAllCommits || prov.IsAgent

	// Branch policy is always present (only fires for agents on protected branches).
	gates := []engine.Gate{
		branchpolicy.New(cfg.Gates.BranchPolicy.Protected, branch, prov.IsAgent),
	}
	if !applyContent {
		return gates, prov
	}

	gates = append(gates, diffbudget.New(cfg.Gates.DiffBudget))

	if cfg.Gates.AIPatterns.Enabled {
		// Approved overrides (trailer/env) skip their rules for this run only —
		// unlike disable_rules, they are per-commit escapes, not policy.
		disables := append(append([]string{}, cfg.Gates.AIPatterns.DisableRules...), overrides...)
		gates = append(gates, aipatterns.New(disables))
	}

	// Scope reads the declared task manifest when present; absent, it is
	// warn-only unless the config requires a manifest.
	var scopePaths []string
	if m, ok, err := taskmanifest.Load(root); err != nil {
		// A malformed manifest is a misconfiguration, not "no manifest" — fail
		// loud rather than silently dropping scope enforcement.
		os.Stderr.WriteString("dwarpal: " + err.Error() + "\n")
	} else if ok {
		scopePaths = m.Paths
	}
	gates = append(gates, scope.New(scopePaths, cfg.Gates.Scope.AllowAlways, cfg.Gates.Scope.RequireTaskManifest))

	return gates, prov
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
