// Package engine orchestrates the gate pipeline.
//
// It runs each enabled gate in order and aggregates their findings. Two
// behaviors are contractual (see the gate-pipeline spec):
//   - Report-everything: with stop_on_first_block false (the M0 default) every
//     gate runs even after one produces a blocking finding.
//   - Fail-closed: a deterministic gate that returns an error is recorded as a
//     GateError, never silently skipped. The caller treats gate errors as
//     blocking in enforce/ci_strict mode.
package engine

import (
	"context"
	"sync"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// RepoIndex is the repo-level context that stateful gates (duplicate-function,
// drift) will need. In M0 no gate uses it, but the Gate signature takes it now
// so adding those gates later never changes the interface. The concrete M0
// implementation is NoIndex.
type RepoIndex interface {
	// Ready reports whether the index has been built. M0's NoIndex returns false.
	Ready() bool
}

// NoIndex is the M0 placeholder RepoIndex.
type NoIndex struct{}

func (NoIndex) Ready() bool { return false }

// Gate is the contract every check implements — the same interface that exec
// plugins will satisfy later, so community gates need not touch the engine.
type Gate interface {
	ID() string
	Run(ctx context.Context, d *gitio.Diff, idx RepoIndex) ([]finding.Finding, error)
}

// GateError records a gate that failed to run. It is surfaced, not swallowed.
type GateError struct {
	Gate string
	Err  error
}

// Result is the aggregate outcome of a pipeline run.
type Result struct {
	Findings   []finding.Finding
	GateErrors []GateError
}

// Blocking reports whether this result should block a commit in enforce mode:
// any error-severity finding, or any gate error (fail-closed).
func (r Result) Blocking() bool {
	if len(r.GateErrors) > 0 {
		return true
	}
	for _, f := range r.Findings {
		if f.Severity.Blocking() {
			return true
		}
	}
	return false
}

// Options tunes a pipeline run.
type Options struct {
	// StopOnFirstBlock ends the run at the first gate that produced a blocking
	// finding (or gate error), instead of the report-everything default.
	StopOnFirstBlock bool
}

// fillDocsURLs gives every finding a working documentation link. Gates may
// set DocsURL themselves; the canonical gate/rule mapping fills the rest, so
// no gate has to repeat it (and none can forget it).
func fillDocsURLs(fs []finding.Finding) []finding.Finding {
	for i := range fs {
		if fs[i].DocsURL == "" {
			fs[i].DocsURL = finding.DocsURL(fs[i].Gate, fs[i].RuleID)
		}
	}
	return fs
}

// Run executes the gates against the diff and aggregates everything.
func Run(ctx context.Context, gates []Gate, d *gitio.Diff, idx RepoIndex) Result {
	return RunWith(ctx, gates, d, idx, Options{})
}

// RunWith executes the gates with explicit options.
//
// In the default report-everything mode gates run CONCURRENTLY (they only read
// the diff, the index, and the work tree) and their results are folded back in
// gate order, so output stays deterministic regardless of completion order.
// StopOnFirstBlock forces sequential execution — its whole point is that later
// gates never run after a block.
func RunWith(ctx context.Context, gates []Gate, d *gitio.Diff, idx RepoIndex, opts Options) Result {
	if opts.StopOnFirstBlock {
		var res Result
		for _, g := range gates {
			fs, err := g.Run(ctx, d, idx)
			if err != nil {
				res.GateErrors = append(res.GateErrors, GateError{Gate: g.ID(), Err: err})
			} else {
				res.Findings = append(res.Findings, fillDocsURLs(fs)...)
			}
			if res.Blocking() {
				break
			}
		}
		return res
	}

	type gateResult struct {
		findings []finding.Finding
		err      error
	}
	results := make([]gateResult, len(gates))
	var wg sync.WaitGroup
	for i, g := range gates {
		wg.Add(1)
		go func(i int, g Gate) {
			defer wg.Done()
			fs, err := g.Run(ctx, d, idx)
			results[i] = gateResult{findings: fs, err: err}
		}(i, g)
	}
	wg.Wait()

	var res Result
	for i, g := range gates {
		if results[i].err != nil {
			res.GateErrors = append(res.GateErrors, GateError{Gate: g.ID(), Err: results[i].err})
			continue
		}
		res.Findings = append(res.Findings, fillDocsURLs(results[i].findings)...)
	}
	return res
}
