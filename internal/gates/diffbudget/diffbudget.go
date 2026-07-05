// Package diffbudget implements Gate 1 — the diff-budget gate.
//
// It blocks changes that are too large to review: total changed lines, changed
// files, or newly added files beyond the configured maxima. Oversized,
// unreviewable diffs are the single most-requested control in practitioner
// threads (PRD §5.2 Gate 1) and the root of most agent-review failures.
//
// Per-path-glob overrides let generated code or lockfiles carry larger budgets.
// A file is measured against the first override whose glob it matches, else the
// global budget — so a diff that is large only because of generated files still
// passes when that path is exempted.
package diffbudget

import (
	"context"
	"fmt"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "diff_budget"

// Gate is the diff-budget gate.
type Gate struct {
	cfg config.DiffBudget
}

// New builds the gate from its configuration.
func New(cfg config.DiffBudget) *Gate { return &Gate{cfg: cfg} }

// ID identifies the gate in findings and config.
func (g *Gate) ID() string { return gateID }

// budget is the effective set of limits applied to one group of files. A zero
// field means "not overridden" and falls back to the global value.
type budget struct {
	label       string
	maxLines    int
	maxFiles    int
	maxNewFiles int
}

// group accumulates the files measured against one budget.
type group struct {
	budget budget
	lines  int
	files  int
	newN   int
}

// Run measures the diff against the global budget and any per-glob overrides,
// emitting one error finding per exceeded budget.
func (g *Gate) Run(_ context.Context, d *gitio.Diff) ([]finding.Finding, error) {
	global := budget{
		label:       "global",
		maxLines:    g.cfg.MaxLines,
		maxFiles:    g.cfg.MaxFiles,
		maxNewFiles: g.cfg.MaxNewFiles,
	}

	// One group per override plus the global group. Override budgets inherit
	// any unset (zero) field from the global budget.
	groups := []*group{{budget: global}}
	for i, o := range g.cfg.Overrides {
		groups = append(groups, &group{budget: budget{
			label:       fmt.Sprintf("override[%d]", i),
			maxLines:    orDefault(o.MaxLines, global.maxLines),
			maxFiles:    orDefault(o.MaxFiles, global.maxFiles),
			maxNewFiles: orDefault(o.MaxNewFiles, global.maxNewFiles),
		}})
	}

	for _, f := range d.Files {
		gi := g.groupFor(f.Path) // 0 = global, else override index+1
		grp := groups[gi]
		grp.lines += f.Added + f.Removed
		grp.files++
		if f.Kind == gitio.KindAdded {
			grp.newN++
		}
	}

	var findings []finding.Finding
	for _, grp := range groups {
		findings = append(findings, grp.check()...)
	}
	return findings, nil
}

// groupFor returns the group index for a path: the first override (1-based)
// whose any glob matches, or 0 for the global group.
func (g *Gate) groupFor(path string) int {
	for i, o := range g.cfg.Overrides {
		for _, glob := range o.Paths {
			if ok, _ := doublestar.Match(glob, path); ok {
				return i + 1
			}
		}
	}
	return 0
}

// check emits a finding for each budget this group exceeds. A budget of 0 or
// less means "no limit" and is skipped.
func (grp *group) check() []finding.Finding {
	var out []finding.Finding
	b := grp.budget
	if b.maxLines > 0 && grp.lines > b.maxLines {
		out = append(out, budgetFinding("max-lines", "changed lines", grp.lines, b.maxLines, b.label))
	}
	if b.maxFiles > 0 && grp.files > b.maxFiles {
		out = append(out, budgetFinding("max-files", "changed files", grp.files, b.maxFiles, b.label))
	}
	if b.maxNewFiles > 0 && grp.newN > b.maxNewFiles {
		out = append(out, budgetFinding("max-new-files", "new files", grp.newN, b.maxNewFiles, b.label))
	}
	return out
}

// budgetFinding builds a finding plus the imperative retry hint an agent acts
// on (PRD §5.4 — the retry_hints example is literally the diff-budget case).
func budgetFinding(ruleID, noun string, actual, allowed int, label string) finding.Finding {
	scope := ""
	if label != "global" {
		scope = fmt.Sprintf(" (%s)", label)
	}
	return finding.Finding{
		Gate:       gateID,
		RuleID:     ruleID,
		Severity:   finding.SeverityError,
		Message:    fmt.Sprintf("%d %s exceeds the %d limit%s", actual, noun, allowed, scope),
		Suggestion: fmt.Sprintf("split this change so each commit stays within %d %s", allowed, noun),
		RetryHint:  fmt.Sprintf("Split this change: %d %s exceeds the %d-%s budget%s. Commit smaller, self-contained changes.", actual, noun, allowed, splitNoun(noun), scope),
	}
}

// splitNoun turns "changed lines" into "line" for the hint's "500-line budget".
func splitNoun(noun string) string {
	switch noun {
	case "changed lines":
		return "line"
	case "changed files":
		return "file"
	case "new files":
		return "new-file"
	default:
		return noun
	}
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
