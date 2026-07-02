// Package intent implements Gate 7 — LLM-based intent verification (PRD
// §5.2 Gate 7, §6 #6).
//
// The gate asks a BYO-key LLM provider whether a diff accomplishes the
// declared task intent, only that intent, and whether anything in it would
// surprise a reviewer. It is advisory: a bad or surprising verdict produces
// warn findings, never errors. Critically, infrastructure failures (provider
// error, timeout) must never block a commit — this is the single documented
// exception to the engine's fail-closed default (see engine.go and PRD §6
// #6), because a third-party API outage is not evidence the diff is unsafe.
package intent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "intent"

// Verdict is the structured result of an intent check.
type Verdict struct {
	AccomplishesIntent bool
	OnlyStatedIntent   bool
	Surprises          []string
	Raw                string // the raw provider response, kept for debugging/audit
}

// Provider verifies a prompt against an LLM and returns a structured verdict.
// Implementations are expected to enforce ctx's deadline themselves (e.g. via
// an HTTP request built with ctx).
type Provider interface {
	Verify(ctx context.Context, prompt string) (Verdict, error)
	Name() string
}

// Gate implements Gate 7 — intent verification.
type Gate struct {
	provider   Provider
	taskIntent string
	timeout    time.Duration
}

// New builds the intent gate. taskIntent is the declared task description
// (from the task manifest); timeout bounds how long the provider call may
// take before the gate fails open.
func New(p Provider, taskIntent string, timeout time.Duration) *Gate {
	return &Gate{provider: p, taskIntent: taskIntent, timeout: timeout}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run asks the configured Provider whether the diff accomplishes the stated
// task intent. Infrastructure failures (provider error, timeout) fail open:
// the gate returns (nil, nil) rather than blocking the commit. This is
// intentional — see the package doc and PRD §6 #6.
func (g *Gate) Run(ctx context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	if g.provider == nil || d.Empty() {
		return nil, nil
	}

	prompt := buildPrompt(g.taskIntent, d)

	callCtx := ctx
	if g.timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, g.timeout)
		defer cancel()
	}

	verdict, err := g.provider.Verify(callCtx, prompt)
	if err != nil {
		// Fail-open: an LLM infra failure (network error, timeout, provider
		// down) is not evidence the diff is unsafe. Never block a commit on
		// it, and never surface it as a gate error either.
		return nil, nil
	}

	return findingsFromVerdict(verdict), nil
}

// buildPrompt renders the task intent and a unified summary of the diff
// (files touched + added lines) into an instruction asking the model to
// judge scope and surface surprises.
func buildPrompt(taskIntent string, d *gitio.Diff) string {
	var b strings.Builder
	b.WriteString("You are verifying that a code diff accomplishes ONLY the stated task intent.\n\n")
	b.WriteString("Task intent:\n")
	if taskIntent == "" {
		b.WriteString("(none declared)\n")
	} else {
		b.WriteString(taskIntent + "\n")
	}
	b.WriteString("\nDiff summary:\n")
	for _, f := range d.Files {
		fmt.Fprintf(&b, "--- %s (%s, +%d/-%d)\n", f.Path, f.Kind, f.Added, f.Removed)
		for _, l := range f.AddedLines {
			fmt.Fprintf(&b, "%d: +%s\n", l.Number, l.Text)
		}
	}
	b.WriteString("\nDoes this diff accomplish the stated intent? Does it accomplish ONLY the stated ")
	b.WriteString("intent? List any changes a reviewer would find surprising or unrelated to the intent.")
	return b.String()
}

// findingsFromVerdict turns a non-clean verdict into warn findings. A clean
// verdict (accomplishes intent, only that intent, no surprises) produces no
// findings.
func findingsFromVerdict(v Verdict) []finding.Finding {
	if v.AccomplishesIntent && v.OnlyStatedIntent && len(v.Surprises) == 0 {
		return nil
	}

	var findings []finding.Finding

	if !v.AccomplishesIntent {
		findings = append(findings, finding.Finding{
			Gate:       gateID,
			RuleID:     "intent-not-accomplished",
			Severity:   finding.SeverityWarn,
			Message:    "the diff does not appear to accomplish the stated task intent",
			Suggestion: "review the diff against the declared task intent, or update the intent if it changed",
			RetryHint:  "The diff doesn't appear to accomplish the stated task intent. Re-check the change against the task description.",
		})
	}

	if !v.OnlyStatedIntent {
		findings = append(findings, finding.Finding{
			Gate:       gateID,
			RuleID:     "intent-scope-exceeded",
			Severity:   finding.SeverityWarn,
			Message:    "the diff appears to do more than the stated task intent",
			Suggestion: "split unrelated changes into a separate commit, or widen the declared intent if they belong",
			RetryHint:  "The diff appears to go beyond the stated task intent. Split unrelated changes into their own commit.",
		})
	}

	for _, s := range v.Surprises {
		findings = append(findings, finding.Finding{
			Gate:       gateID,
			RuleID:     "intent-surprise",
			Severity:   finding.SeverityWarn,
			Message:    fmt.Sprintf("surprising change: %s", s),
			Suggestion: "confirm this change is intentional and expected by a reviewer",
			RetryHint:  fmt.Sprintf("A reviewer would find this surprising: %s. Confirm it's intentional or remove it.", s),
		})
	}

	return findings
}
