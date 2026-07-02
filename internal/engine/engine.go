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

// Run executes the gates against the diff and aggregates everything.
func Run(ctx context.Context, gates []Gate, d *gitio.Diff, idx RepoIndex) Result {
	var res Result
	for _, g := range gates {
		fs, err := g.Run(ctx, d, idx)
		if err != nil {
			res.GateErrors = append(res.GateErrors, GateError{Gate: g.ID(), Err: err})
			continue
		}
		res.Findings = append(res.Findings, fs...)
	}
	return res
}
