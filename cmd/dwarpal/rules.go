package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/aipatterns"
)

func newRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "List the active gates and rules and their source",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRules()
		},
	}
	cmd.AddCommand(newRulesTestCmd())
	return cmd
}

func newRulesTestCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Verify each built-in rule against its positive/negative examples",
		Long: "test checks that every built-in ai_patterns rule flags the code it should and " +
			"stays silent on the code it shouldn't — the rule set as a tested spec. A negative " +
			"example that wrongly matches means a rule is too broad (a false-positive-budget risk); " +
			"a rule with no examples is an untested gap. Exits non-zero if any rule fails, so it " +
			"can gate rule changes in CI.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRulesTest(jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the results as JSON")
	return cmd
}

func runRulesTest(jsonOut bool) error {
	checks := aipatterns.CheckExamples()

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(checks); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
	} else {
		printRulesTest(checks)
	}

	for _, c := range checks {
		if !c.OK() {
			return &exitError{code: 1, msg: "some rules failed their example tests"}
		}
	}
	return nil
}

func printRulesTest(checks []aipatterns.RuleCheck) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "RULE\tSEVERITY\t+EX\t-EX\tSTATUS")
	passed := 0
	for _, c := range checks {
		status := "pass"
		if c.OK() {
			passed++
		} else {
			status = "FAIL"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n", c.RuleID, c.Severity, c.Positives, c.Negatives, status)
	}
	w.Flush()
	for _, c := range checks {
		for _, f := range c.Failures {
			fmt.Printf("  • %s: %s\n", c.RuleID, f)
		}
	}
	fmt.Printf("\n%d/%d rules pass their example tests.\n", passed, len(checks))
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
