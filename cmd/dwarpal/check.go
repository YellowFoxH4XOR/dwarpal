package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
	"github.com/YellowFoxH4XOR/dwarpal/internal/report"
)

func newCheckCmd() *cobra.Command {
	var (
		jsonOut  bool
		sarifOut bool
		rangeArg string
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run the gate pipeline against staged changes (or a commit range)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCheck(jsonOut, sarifOut, rangeArg)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON (stdout only)")
	cmd.Flags().BoolVar(&sarifOut, "sarif", false, "emit SARIF 2.1.0 for CI annotation (stdout only)")
	cmd.Flags().StringVar(&rangeArg, "range", "", "check a commit range instead of the staging area, e.g. HEAD~1..HEAD")
	return cmd
}

func runCheck(jsonOut, sarifOut bool, rangeArg string) error {
	if !gitAvailable() {
		return &exitError{code: 2, msg: gitio.ErrGitNotFound.Error()}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "not a git repository (run inside a repo)"}
	}

	cfg, err := config.Load(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	ex := gitio.NewExtractor(root)
	var diff *gitio.Diff
	if rangeArg != "" {
		diff, err = ex.Range(rangeArg)
	} else {
		diff, err = ex.Staged()
	}
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	gates, _, idx := buildGates(root, cfg)
	res := engine.Run(context.Background(), gates, diff, idx)

	blocking := res.Blocking() && cfg.Mode != config.ModeWarn
	in := report.Input{
		Result:     resultString(cfg.Mode, res, diff),
		Findings:   res.Findings,
		GateErrors: res.GateErrors,
	}

	// In --json mode stdout carries only the JSON document; everything human
	// stays on stderr. In TTY mode the report goes to stdout.
	if sarifOut {
		if err := report.SARIF(os.Stdout, in); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
	} else if jsonOut {
		if err := report.JSON(os.Stdout, in); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
	} else {
		if diff.Empty() {
			fmt.Fprintln(os.Stdout, "Dwarpal: nothing staged to check.")
		} else if err := report.TTY(os.Stdout, in); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
	}

	if blocking {
		return &exitError{code: 1}
	}
	return nil
}

// resultString maps mode + outcome to the JSON result contract.
func resultString(mode config.Mode, res engine.Result, diff *gitio.Diff) string {
	if diff.Empty() || (len(res.Findings) == 0 && len(res.GateErrors) == 0) {
		return report.ResultPassed
	}
	if mode == config.ModeWarn {
		return report.ResultWarned
	}
	if res.Blocking() {
		return report.ResultBlocked
	}
	return report.ResultPassed
}
