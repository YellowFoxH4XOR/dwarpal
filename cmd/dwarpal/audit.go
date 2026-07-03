package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/audit"
)

func newAuditCmd() *cobra.Command {
	opts := audit.Defaults()
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Self-calibrate rules against git history: which flags do humans actually act on?",
		Long: "audit replays recent commits through the ai_patterns gate and measures, per rule, " +
			"the fraction of flagged lines a human later rewrote or removed (the \"acted-on rate\"). " +
			"A rule people act on catches real problems; a rule whose flags survive untouched is noise. " +
			"It is deterministic and offline — no LLM, no network — and, like `dwarpal analyze`, it only " +
			"prints facts and never modifies .dwarpal.yml. Use it to find and retire noisy rules.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAudit(opts, jsonOut)
		},
	}
	cmd.Flags().IntVar(&opts.Window, "window", opts.Window, "most-recent non-merge commits to replay")
	cmd.Flags().IntVar(&opts.MinSamples, "min-samples", opts.MinSamples, "minimum flags before a rule is advised on")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the calibration as JSON (for agent consumption)")
	return cmd
}

func runAudit(opts audit.Options, jsonOut bool) error {
	if !gitAvailable() {
		return &exitError{code: 2, msg: "system git executable is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}
	rep, err := audit.Run(root, opts)
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
	printAudit(rep)
	return nil
}

func printAudit(rep *audit.Report) {
	fmt.Printf("Dwarpal audit — rule self-calibration over %d commits (deterministic, no network)\n\n",
		rep.CommitsScanned)
	if len(rep.Rules) == 0 {
		fmt.Println("No ai_patterns findings in the scanned history — nothing to calibrate.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "RULE\tSEVERITY\tSAMPLES\tACTED-ON\tRECOMMENDATION")
	for _, r := range rep.Rules {
		rec := r.Recommendation
		if rec == "" {
			rec = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%.0f%%\t%s\n",
			r.RuleID, r.CurrentSeverity, r.Samples, r.ActedOnRate*100, rec)
	}
	w.Flush()

	fmt.Println("\nActed-on = the flagged line was later rewritten or removed by a human.")
	fmt.Println("Low acted-on over enough samples = noise; consider demoting. High = signal.")
	fmt.Println("Advisory only — Dwarpal never edits .dwarpal.yml; you or your agent decide.")
}
