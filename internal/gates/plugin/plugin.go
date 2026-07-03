// Package plugin implements Gate 8 — the exec-plugin gate.
//
// It runs an external command (semgrep, gitleaks, osv-scanner, a custom script)
// against the change. A nonzero exit is treated as findings. This turns Dwarpal
// into the orchestrator of a team's existing tools at the pre-commit boundary —
// an adoption lever, not lock-in (PRD §5.2 Gate 8).
//
// The same Gate interface the built-in gates implement is exposed here, so a
// plugin is indistinguishable from a native gate to the engine.
package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// Gate runs one configured external command.
type Gate struct {
	name    string
	command string
	when    []string // path globs; empty = always run
	dir     string   // working directory (repo root)
}

// New builds an exec-plugin gate. command is run via `sh -c`, so shell
// pipelines like "semgrep scan --json" work. when limits the gate to changes
// touching matching paths; empty means always run.
func New(name, command string, when []string, dir string) *Gate {
	return &Gate{name: name, command: command, when: when, dir: dir}
}

// ID namespaces the plugin so its findings are attributable.
func (g *Gate) ID() string { return "plugin/" + g.name }

// Run executes the command when it applies. Exit 0 = pass (no findings);
// nonzero = one finding carrying the tool's output. An inability to start the
// command is returned as an error so the engine fails closed on it.
func (g *Gate) Run(ctx context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	if !g.applies(d) {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", g.command)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil, nil // exit 0: the tool found nothing to block
	}

	var exitErr *exec.ExitError
	if !asExit(err, &exitErr) {
		// Command could not be started/found — fail closed with a clear error.
		return nil, fmt.Errorf("plugin %q failed to run: %w", g.name, err)
	}

	// Structured path (#44): tools that emit JSON (gitleaks, semgrep) get their
	// findings mapped individually with file:line, instead of one blob finding.
	if fs, ok := ParseToolJSON(g.name, out); ok {
		return fs, nil
	}

	return []finding.Finding{{
		Gate:       g.ID(),
		RuleID:     "exit-nonzero",
		Severity:   finding.SeverityError,
		Message:    fmt.Sprintf("%s reported findings (exit %d)", g.name, exitErr.ExitCode()),
		Suggestion: firstLines(string(out), 20),
		RetryHint:  fmt.Sprintf("The %q check failed. Review its output and fix the reported issues before committing.", g.name),
	}}, nil
}

// applies reports whether any changed file matches the when globs.
func (g *Gate) applies(d *gitio.Diff) bool {
	if len(g.when) == 0 {
		return true
	}
	for _, f := range d.Files {
		for _, glob := range g.when {
			if ok, _ := doublestar.Match(glob, f.Path); ok {
				return true
			}
		}
	}
	return false
}

func asExit(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// firstLines returns at most n lines of s, for a readable suggestion snippet.
func firstLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = append(lines[:n], "… (truncated)")
	}
	return strings.Join(lines, "\n")
}
