package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/aipatterns"
)

func newRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "List the active gates and rules and their source",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRules()
		},
	}
}

// runRules prints the gates that would run given the current config, so a user
// can see what is enforced and where each setting comes from (PRD §5.1).
func runRules() error {
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}
	cfg, err := config.Load(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	fmt.Printf("mode: %s   apply_gates_to: %s\n\n", cfg.Mode, cfg.Provenance.ApplyGatesTo)
	fmt.Println("Gates:")
	fmt.Printf("  diff_budget    error   %d lines / %d files / %d new files\n",
		cfg.Gates.DiffBudget.MaxLines, cfg.Gates.DiffBudget.MaxFiles, cfg.Gates.DiffBudget.MaxNewFiles)
	fmt.Printf("  branch_policy  error   protected: %v\n", cfg.Gates.BranchPolicy.Protected)
	if cfg.Gates.AIPatterns.Enabled {
		fmt.Println("  ai_patterns    error   rules:")
		disabled := map[string]bool{}
		for _, id := range cfg.Gates.AIPatterns.DisableRules {
			disabled[id] = true
		}
		for _, id := range aipatterns.RuleIDs() {
			state := "on"
			if disabled[id] {
				state = "disabled"
			}
			fmt.Printf("                           - %-40s %s\n", id, state)
		}
	}
	fmt.Printf("  scope          error   require_manifest: %v\n", cfg.Gates.Scope.RequireTaskManifest)
	if cfg.Gates.Duplicate.Enabled {
		fmt.Printf("  duplicate      warn    no-duplicate-function (Go), threshold %.2f\n", cfg.Gates.Duplicate.Threshold)
	} else {
		fmt.Println("  duplicate      off     enable with gates.duplicate.enabled (builds the repo index)")
	}
	if cfg.Gates.ConventionDrift.Enabled {
		fmt.Printf("  drift          %-6s  convention drift (Go)\n", cfg.Gates.ConventionDrift.Severity)
	}
	if cfg.Gates.DiffCoverage.Artifact != "" {
		fmt.Printf("  diff_coverage  error   %.0f%% on changed lines (artifact: %s)\n",
			cfg.Gates.DiffCoverage.MinPercent, cfg.Gates.DiffCoverage.Artifact)
	}
	if cfg.Gates.IntentCheck.Enabled {
		fmt.Printf("  intent         warn    provider: %s (fail-open)\n", cfg.Gates.IntentCheck.Provider)
	}
	for _, p := range cfg.Gates.Plugins {
		fmt.Printf("  plugin/%-8s error   exec: %s\n", p.Name, p.Exec)
	}
	return nil
}
