package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/census"
	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/report"
)

func newCensusCmd() *cobra.Command {
	var jsonOut, update, check, list bool
	cmd := &cobra.Command{
		Use:   "census",
		Short: "Whole-repo decay ratchet: block changes that INCREASE dead/duplicate/unused counts",
		Long: "census runs configured whole-repo detectors (dead code, unused symbols, duplication) " +
			"and counts what they find across the ENTIRE repository — the cumulative decay a diff-scoped " +
			"gate like `dwarpal check` structurally cannot see. On its own it prints a report. With " +
			"--update-baseline it records the current counts as an accepted baseline (committed to the " +
			"repo). With --check it fails only when a count went UP versus that baseline, naming the new " +
			"items: existing debt is grandfathered, new debt is blocked, the number can only ratchet down. " +
			"Detectors are external tools you install; a configured-but-missing detector fails the check " +
			"loudly rather than passing as zero.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCensus(jsonOut, update, check, list)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON (for agent consumption)")
	cmd.Flags().BoolVar(&update, "update-baseline", false, "write the current counts as the accepted baseline")
	cmd.Flags().BoolVar(&check, "check", false, "fail (exit 1) if any count rose above the baseline")
	cmd.Flags().BoolVar(&list, "list", false, "list the available detectors and exit")
	return cmd
}

func runCensus(jsonOut, update, check, list bool) error {
	if list {
		printDetectorList()
		return nil
	}
	if update && check {
		return &exitError{code: 2, msg: "choose one of --update-baseline or --check, not both"}
	}
	if !gitAvailable() {
		return &exitError{code: 2, msg: "system git executable is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}
	cfg, err := config.Load(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	if len(cfg.Census.Detectors) == 0 {
		fmt.Fprintln(os.Stderr, "census: no detectors configured — add `census: { detectors: [...] }` to "+config.Filename)
		return nil
	}
	baselinePath := cfg.Census.Baseline
	if baselinePath == "" {
		baselinePath = census.DefaultBaselinePath
	}

	rep, err := census.Run(context.Background(), root, cfg.Census.Detectors)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	switch {
	case update:
		return runUpdate(root, baselinePath, rep, jsonOut)
	case check:
		return runCheck2(root, baselinePath, rep, jsonOut)
	default:
		return runReport(rep, jsonOut)
	}
}

// runReport prints the current whole-repo counts (no baseline comparison).
func runReport(rep *census.Report, jsonOut bool) error {
	if jsonOut {
		return emitJSON(rep)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "DETECTOR\tSCOPE\tCOUNT\tSTATUS")
	for _, d := range rep.Detectors {
		status := "ok"
		if d.Skipped {
			status = "SKIPPED — " + d.SkipReason
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", d.Name, d.Scope, d.Count, status)
	}
	w.Flush()
	fmt.Println("\nRun `dwarpal census --update-baseline` to accept these counts, then `--check` in CI to block increases.")
	return nil
}

// runUpdate writes the accepted baseline.
func runUpdate(root, path string, rep *census.Report, jsonOut bool) error {
	if err := census.WriteBaseline(root, path, rep); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	if skipped := rep.SkippedConfigured(); len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "census: warning — %d detector(s) skipped (not installed), excluded from the baseline: %s\n",
			len(skipped), strings.Join(skipped, ", "))
	}
	if jsonOut {
		return emitJSON(rep)
	}
	fmt.Printf("census: wrote baseline for %d detector(s) to %s\n", len(rep.Detectors)-len(rep.SkippedConfigured()), path)
	return nil
}

// runCheck2 is the ratchet: it blocks increases over the committed baseline.
// A configured-but-missing detector is fatal (exit 2), never a silent pass —
// a ratchet you could not run is not a ratchet that passed.
func runCheck2(root, path string, rep *census.Report, jsonOut bool) error {
	if skipped := rep.SkippedConfigured(); len(skipped) > 0 {
		return &exitError{code: 2, msg: fmt.Sprintf(
			"cannot verify the ratchet — detector(s) not installed: %s. Install them or remove them from %s.",
			strings.Join(skipped, ", "), config.Filename)}
	}
	base, err := census.LoadBaseline(root, path)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	if base == nil {
		return &exitError{code: 2, msg: fmt.Sprintf(
			"no baseline at %s — run `dwarpal census --update-baseline` first", path)}
	}

	findings := census.Diff(base, rep)
	result := report.ResultPassed
	if len(findings) > 0 {
		result = report.ResultBlocked
	}
	in := report.Input{Result: result, Findings: findings}

	if jsonOut {
		if err := report.JSON(os.Stdout, in); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
	} else if err := report.TTY(os.Stdout, in); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	if len(findings) > 0 {
		return &exitError{code: 1}
	}
	return nil
}

func printDetectorList() {
	names := census.Names()
	sort.Strings(names)
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "DETECTOR\tSCOPE\tCOMMAND\tINSTALLED")
	for _, n := range names {
		d, _ := census.Lookup(n)
		installed := "no"
		if d.Available() {
			installed = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.Name, d.Scope, d.Command, installed)
	}
	w.Flush()
}

func emitJSON(rep *census.Report) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	return nil
}
