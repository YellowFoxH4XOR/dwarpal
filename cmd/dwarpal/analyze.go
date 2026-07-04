package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/analyze"
)

func newAnalyzeCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Measure the repo and print facts an agent can use to author .dwarpal.yml",
		Long: "analyze measures this repository — its conventions, commit-size history, " +
			"coverage artifacts, security tools, and layering — and prints the facts. " +
			"It makes no network calls and never touches your config or source (it only " +
			"warms the gitignored convention cache the gates already use). Feed the output to your coding " +
			"agent (Claude Code / Codex / OpenCode / Pi) and have it author a .dwarpal.yml " +
			"consistent with your codebase. `dwarpal agent setup <tool>` teaches the agent to do this.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAnalyze(jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the measured facts as JSON (for agent consumption)")
	return cmd
}

func runAnalyze(jsonOut bool) error {
	if !gitAvailable() {
		return &exitError{code: 2, msg: "system git executable is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}
	rep, err := analyze.Run(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
		return nil
	}

	printReport(rep)
	return nil
}

func printReport(r *analyze.Report) {
	fmt.Println("Dwarpal repo analysis (deterministic, no network) — facts for authoring .dwarpal.yml")
	fmt.Println()

	fmt.Printf("Languages: %v\n", orNone(r.Languages))
	fmt.Printf("Suggested diff_budget.max_lines: %d  (%s", r.DiffBudget.MaxLines, r.DiffBudget.Basis)
	if r.DiffBudget.SampleCount > 0 {
		fmt.Printf(", %d commits sampled", r.DiffBudget.SampleCount)
	}
	fmt.Println(")")
	// Only shown when the budget was actually fitted to history (populated
	// distribution); the thin-history fallback has no meaningful percentiles.
	if r.DiffBudget.MaxSeen > 0 {
		fmt.Printf("  commit-size distribution: median %d, p75 %d, p90 %d, max %d changed lines\n",
			r.DiffBudget.MedianLines, r.DiffBudget.P75Lines, r.DiffBudget.P90Lines, r.DiffBudget.MaxSeen)
	}

	if len(r.Conventions) > 0 {
		fmt.Println("\nConventions (drift baselines):")
		langs := make([]string, 0, len(r.Conventions))
		for l := range r.Conventions {
			langs = append(langs, l)
		}
		sort.Strings(langs)
		for _, l := range langs {
			c := r.Conventions[l]
			fmt.Printf("  %s:\n", l)
			if c.DominantImportForm != "" {
				fmt.Printf("    imports: %s (%.0f%%)\n", c.DominantImportForm, c.ImportShare*100)
			}
			if c.DominantErrorIdiom != "" {
				fmt.Printf("    error idiom: %s (%.0f%%)\n", c.DominantErrorIdiom, c.ErrorIdiomShare*100)
			}
			if c.DominantNaming != "" {
				fmt.Printf("    naming: %s\n", c.DominantNaming)
			}
			if c.Funcs > 0 {
				fmt.Printf("    %d functions, avg %d lines, %d snake_case\n", c.Funcs, c.AvgFuncLines, c.SnakeCaseFuncs)
			}
		}
	}

	if r.CoverageArtifact != "" {
		fmt.Printf("\nCoverage artifact found: %s  → wire gates.diff_coverage.artifact\n", r.CoverageArtifact)
	}
	if len(r.SecurityTools) > 0 {
		fmt.Printf("Security tools in repo: %v  → wire as gates.plugins\n", r.SecurityTools)
	}
	if len(r.BranchPrefixes) > 0 {
		fmt.Printf("Branch prefixes in use: %v  → consider provenance.branch_prefixes\n", r.BranchPrefixes)
	}
	if len(r.LayeringHints) > 0 {
		fmt.Println("\nLayering signals (candidate architecture_rules):")
		for _, h := range r.LayeringHints {
			fmt.Printf("  - %s\n", h)
		}
	}

	fmt.Println("\nNext: have your agent author .dwarpal.yml from these facts + the codebase.")
	fmt.Println("      `dwarpal agent setup <claude-code|codex|opencode|pi>` wires that workflow;")
	fmt.Println("      then: \"set up Dwarpal for this repo\". Verify with `dwarpal rules`.")
}

func orNone(s []string) any {
	if len(s) == 0 {
		return "none detected"
	}
	return s
}
