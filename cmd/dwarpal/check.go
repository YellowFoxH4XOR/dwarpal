package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
	"github.com/YellowFoxH4XOR/dwarpal/internal/provenance"
	"github.com/YellowFoxH4XOR/dwarpal/internal/report"
)

func newCheckCmd() *cobra.Command {
	var (
		jsonOut   bool
		sarifOut  bool
		rangeArg  string
		diffFile  string
		perCommit bool
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run the gate pipeline against staged changes (or a commit range)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCheck(jsonOut, sarifOut, rangeArg, diffFile, perCommit)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON (stdout only)")
	cmd.Flags().BoolVar(&jsonOut, "explain-for-agent", false, "alias of --json: block output an agent can consume to self-correct")
	cmd.Flags().BoolVar(&sarifOut, "sarif", false, "emit SARIF 2.1.0 for CI annotation (stdout only)")
	cmd.Flags().StringVar(&rangeArg, "range", "", "check a commit range instead of the staging area, e.g. HEAD~1..HEAD")
	cmd.Flags().StringVar(&diffFile, "diff", "", "check a unified-diff patch file instead of the staging area")
	cmd.Flags().BoolVar(&perCommit, "per-commit", false, "with --range: evaluate each commit's diff separately (budgets are per commit, PRD §5.2)")
	return cmd
}

func runCheck(jsonOut, sarifOut bool, rangeArg, diffFile string, perCommit bool) error {
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
	switch {
	case diffFile != "":
		diff, err = gitio.FromPatchFile(diffFile)
	case rangeArg != "":
		diff, err = ex.Range(rangeArg)
	default:
		diff, err = ex.Staged()
	}
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	gates, prov, idx := buildGates(root, cfg, collectOverrides(root, rangeArg))

	var res engine.Result
	if perCommit && rangeArg != "" {
		// Budgets are defined PER COMMIT (PRD §5.2 Gate 1): a range of three
		// compliant commits must not fail because their sum exceeds one
		// commit's budget. Evaluate each commit's own diff; merge commits are
		// skipped (their content arrived via their parents, same rule as the
		// pre-push marker check).
		res, err = runPerCommit(ex, root, gates, idx, cfg, rangeArg)
		if err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
	} else {
		res = engine.RunWith(context.Background(), gates, diff, idx,
			engine.Options{StopOnFirstBlock: cfg.StopOnFirstBlock})
	}

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
	// Passing agent check: record provenance as a git note for later
	// git-blame forensics (#19). Best-effort — never affects the verdict.
	if prov.IsAgent {
		_ = provenance.AttachNote(root, prov)
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

// runPerCommit evaluates every non-merge commit in the range independently
// and merges the results. Findings keep their file:line; a commit reference
// is appended to the message so multi-commit output stays attributable.
func runPerCommit(ex *gitio.Extractor, root string, gates []engine.Gate, idx engine.RepoIndex, cfg config.Config, rangeArg string) (engine.Result, error) {
	shas, err := revList(root, rangeArg)
	if err != nil {
		return engine.Result{}, err
	}
	var merged engine.Result
	for _, sha := range shas {
		diff, err := ex.Range(sha + "~1.." + sha)
		if err != nil {
			return engine.Result{}, err
		}
		res := engine.RunWith(context.Background(), gates, diff, idx,
			engine.Options{StopOnFirstBlock: cfg.StopOnFirstBlock})
		for i := range res.Findings {
			res.Findings[i].Message += " (commit " + sha[:7] + ")"
		}
		merged.Findings = append(merged.Findings, res.Findings...)
		merged.GateErrors = append(merged.GateErrors, res.GateErrors...)
	}
	return merged, nil
}

// revList returns the range's non-merge commits, oldest first.
func revList(root, rangeArg string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", "--no-merges", "--reverse", rangeArg)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list %s: %w", rangeArg, err)
	}
	var shas []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			shas = append(shas, l)
		}
	}
	return shas, nil
}
