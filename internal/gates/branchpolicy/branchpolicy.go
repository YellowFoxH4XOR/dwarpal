// Package branchpolicy implements the branch-policy half of Gate 2.
//
// It blocks agent-authored commits to protected branches (main, release/*),
// pushing agent work onto prefixed branches where it can be reviewed (PRD §5.2
// Gate 2, failure mode 8: direct commits to shared branches).
//
// Branch name and agent provenance are commit context, not part of the Diff, so
// they are injected at construction — keeping the Gate.Run signature stable.
package branchpolicy

import (
	"context"
	"fmt"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "branch_policy"

// Gate enforces protected-branch policy for agent commits.
type Gate struct {
	protected []string // branch globs, e.g. ["main", "release/*"]
	branch    string   // current branch
	isAgent   bool     // whether this change is agent-authored
}

// New builds the gate with the protected globs and the current commit context.
func New(protected []string, branch string, isAgent bool) *Gate {
	return &Gate{protected: protected, branch: branch, isAgent: isAgent}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run blocks when an agent-authored change targets a protected branch. Human
// commits are never blocked by this gate.
func (g *Gate) Run(_ context.Context, _ *gitio.Diff) ([]finding.Finding, error) {
	if !g.isAgent {
		return nil, nil
	}
	for _, glob := range g.protected {
		if ok, _ := doublestar.Match(glob, g.branch); ok {
			return []finding.Finding{{
				Gate:       gateID,
				RuleID:     "protected-branch",
				Severity:   finding.SeverityError,
				Message:    fmt.Sprintf("agent commit to protected branch %q", g.branch),
				Suggestion: "move this work to an agent/* branch and open a PR",
				RetryHint:  fmt.Sprintf("Do not commit agent work directly to %q. Create an agent/<task> branch and commit there.", g.branch),
			}}, nil
		}
	}
	return nil, nil
}
