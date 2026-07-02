// Package drift implements Gate 6 — convention drift (PRD §5.2 Gate 6).
//
// It scores added Go functions against the repo's own convention fingerprint
// (from repoindex) and flags outliers: naming style that bucks the repo norm,
// or functions far longer than the repo's typical size. This catches
// fluent-but-foreign agent code (failure mode 6).
//
// The gate is honest about being heuristic: it ships severity: info by default.
package drift

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
	"github.com/YellowFoxH4XOR/dwarpal/internal/repoindex"
)

const gateID = "convention_drift"

// Gate flags convention outliers in added code.
type Gate struct {
	root     string
	severity finding.Severity
}

// New builds the gate. severity defaults to info when empty.
func New(root string, severity finding.Severity) *Gate {
	if severity == "" {
		severity = finding.SeverityInfo
	}
	return &Gate{root: root, severity: severity}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run scores each added Go function against the repo fingerprint.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, idx engine.RepoIndex) ([]finding.Finding, error) {
	index, ok := idx.(*repoindex.Index)
	if !ok || !index.Ready() {
		return nil, nil
	}
	conv := index.Conventions
	// Need a baseline to compare against.
	if conv.Funcs < 5 {
		return nil, nil
	}
	snakeRatio := float64(conv.SnakeCaseFuncs) / float64(conv.Funcs)
	avg := conv.AvgFuncLines()

	var findings []finding.Finding
	for _, f := range d.Files {
		if !strings.HasSuffix(f.Path, ".go") || len(f.AddedLines) == 0 {
			continue
		}
		added := map[int]bool{}
		for _, ln := range f.AddedLines {
			added[ln.Number] = true
		}
		src, err := os.ReadFile(filepath.Join(g.root, f.Path))
		if err != nil {
			continue
		}
		for _, fn := range repoindex.FunctionsInSource(f.Path, src) {
			if !touches(fn, added) {
				continue
			}
			// Naming drift: snake_case where the repo overwhelmingly isn't.
			if strings.Contains(fn.Name, "_") && snakeRatio < 0.1 {
				findings = append(findings, g.finding(f.Path, fn.StartLine,
					"naming-style",
					fmt.Sprintf("function %s uses snake_case; the repo overwhelmingly uses Go camelCase", fn.Name),
					"rename to match the repo's naming convention"))
			}
			// Size drift: much longer than the repo norm.
			length := fn.EndLine - fn.StartLine + 1
			if avg > 0 && float64(length) > 3*avg {
				findings = append(findings, g.finding(f.Path, fn.StartLine,
					"function-size",
					fmt.Sprintf("function %s is %d lines; the repo average is %.0f", fn.Name, length, avg),
					"consider splitting this function to match the repo's typical size"))
			}
		}
	}
	return findings, nil
}

func (g *Gate) finding(file string, line int, rule, msg, suggestion string) finding.Finding {
	return finding.Finding{
		Gate:       gateID,
		RuleID:     rule,
		Severity:   g.severity,
		File:       file,
		Line:       line,
		Message:    msg,
		Suggestion: suggestion,
	}
}

func touches(fn repoindex.FuncInfo, added map[int]bool) bool {
	for ln := fn.StartLine; ln <= fn.EndLine; ln++ {
		if added[ln] {
			return true
		}
	}
	return false
}
