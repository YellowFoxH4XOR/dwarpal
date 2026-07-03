package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/audit"
	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
)

func newAuditCmd() *cobra.Command {
	opts := audit.Defaults()
	var jsonOut, apply bool
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Self-calibrate rules against git history: which flags do humans actually act on?",
		Long: "audit replays recent commits through the ai_patterns gate and measures, per rule, " +
			"the fraction of flagged lines a human later rewrote or removed (the \"acted-on rate\"). " +
			"A rule people act on catches real problems; a rule whose flags survive untouched is noise. " +
			"It is deterministic and offline — no LLM, no network — and, like `dwarpal analyze`, it only " +
			"prints facts and never modifies .dwarpal.yml. Use it to find and retire noisy rules.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAudit(opts, jsonOut, apply)
		},
	}
	cmd.Flags().IntVar(&opts.Window, "window", opts.Window, "most-recent non-merge commits to replay")
	cmd.Flags().IntVar(&opts.MinSamples, "min-samples", opts.MinSamples, "minimum flags before a rule is advised on")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the calibration as JSON (for agent consumption)")
	cmd.Flags().BoolVar(&apply, "apply", false, "write DEMOTE recommendations to .dwarpal.yml rule_overrides (never auto-promotes)")
	return cmd
}

func runAudit(opts audit.Options, jsonOut, apply bool) error {
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

	if apply {
		return applyDemotions(root, rep)
	}
	return nil
}

// applyDemotions writes only the safe demotions (error → warn) to the config.
// Promotions are surfaced for manual review but never auto-applied, because
// auto-promoting a rule to hard-block on this fuzzy signal is the worst case.
func applyDemotions(root string, rep *audit.Report) error {
	demotes := map[string]string{}
	promoteReviews := 0
	for _, r := range rep.Rules {
		if r.Demote {
			demotes[r.Gate+"/"+r.RuleID] = "warn"
		} else if r.Recommendation != "" {
			promoteReviews++
		}
	}
	fmt.Println()
	if len(demotes) == 0 {
		fmt.Println("--apply: no safe demotions to write.")
	} else {
		if err := config.PatchRuleOverrides(root, demotes); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
		fmt.Printf("--apply: wrote %d demotion(s) to %s rule_overrides.\n", len(demotes), config.Filename)
	}
	if promoteReviews > 0 {
		fmt.Printf("--apply: %d rule(s) suggested for PROMOTION were NOT applied — review them by hand.\n", promoteReviews)
	}
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
