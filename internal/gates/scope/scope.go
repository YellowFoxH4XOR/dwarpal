// Package scope implements Gate 4 — scope enforcement.
//
// An agent (or human) declares the task's intended paths; the gate blocks
// changes to files outside that set. This targets scope creep — the agent
// modifying files that have nothing to do with the task (PRD §1 failure mode 2,
// §5.2 Gate 4). Always-allow globs (lockfiles, snapshots) are exempt.
//
// When no task manifest is declared the gate is warn-only by default
// (configurable to block), so it never surprises a user who hasn't opted in.
package scope

import (
	"context"
	"fmt"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "scope"

// implicitAllow is always in scope: Dwarpal's own config and manifest files,
// which a task legitimately touches while declaring or adjusting its scope.
var implicitAllow = []string{".dwarpal.yml", ".dwarpal-task.yml", ".dwarpal/**"}

// Gate enforces the declared task scope.
type Gate struct {
	paths           []string // declared in-scope globs; empty = no manifest
	allowAlways     []string // always-permitted globs (lockfiles, snapshots)
	requireManifest bool     // block when no manifest is present
}

// New builds the scope gate. paths is the declared in-scope set (empty means no
// manifest); allowAlways is always permitted; requireManifest blocks when no
// manifest exists instead of warning.
func New(paths, allowAlways []string, requireManifest bool) *Gate {
	return &Gate{paths: paths, allowAlways: allowAlways, requireManifest: requireManifest}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run flags each changed file outside the declared scope.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	if len(g.paths) == 0 {
		// No manifest. Block only if configured to require one.
		if g.requireManifest {
			return []finding.Finding{{
				Gate:       gateID,
				RuleID:     "no-task-manifest",
				Severity:   finding.SeverityError,
				Message:    "no task manifest declares the intended scope of this change",
				Suggestion: "declare intent with `dwarpal task \"<id>\" --paths <globs>` or a .dwarpal-task.yml",
				RetryHint:  "Declare the task scope before committing: which paths is this change allowed to touch?",
			}}, nil
		}
		return nil, nil // warn-only default: nothing to enforce
	}

	var findings []finding.Finding
	for _, f := range d.Files {
		if g.matchesAny(f.Path, implicitAllow) || g.matchesAny(f.Path, g.allowAlways) || g.matchesAny(f.Path, g.paths) {
			continue
		}
		findings = append(findings, finding.Finding{
			Gate:       gateID,
			RuleID:     "out-of-scope",
			Severity:   finding.SeverityError,
			File:       f.Path,
			Message:    fmt.Sprintf("%s is outside the declared task scope", f.Path),
			Suggestion: "commit this file separately, or widen the task's declared paths if it belongs",
			RetryHint:  fmt.Sprintf("File %s is outside the declared task scope. Split unrelated changes into their own commit.", f.Path),
		})
	}
	return findings, nil
}

func (g *Gate) matchesAny(path string, globs []string) bool {
	for _, glob := range globs {
		if ok, _ := doublestar.Match(glob, path); ok {
			return true
		}
	}
	return false
}
