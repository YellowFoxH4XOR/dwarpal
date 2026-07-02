// Package duplicate implements the no-duplicate-function rule (PRD §5.2 Gate 3).
//
// For each function a change adds or edits, it compares the function's token
// shingles against the repo's existing function inventory (repoindex) and flags
// near-duplicates above a similarity threshold — catching an agent re-solving a
// problem the codebase already solved. Heuristic, so warn severity by default.
//
// v1 covers Go (via repoindex's go/parser inventory); other languages are a
// future change once tree-sitter grammars land.
package duplicate

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

const gateID = "ai_patterns" // reported as ai_patterns/no-duplicate-function

// Gate detects near-duplicate added functions.
type Gate struct {
	root      string
	threshold float64
}

// New builds the gate. threshold is the Jaccard cutoff (e.g. 0.85).
func New(root string, threshold float64) *Gate {
	return &Gate{root: root, threshold: threshold}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run compares each added/edited Go function against the repo index.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, idx engine.RepoIndex) ([]finding.Finding, error) {
	index, ok := idx.(*repoindex.Index)
	if !ok || !index.Ready() {
		return nil, nil // no index available — skip rather than error
	}

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
			if !touchesAdded(fn, added) {
				continue
			}
			if match, score := bestMatch(fn, index, f.Path, g.threshold); match != nil {
				findings = append(findings, finding.Finding{
					Gate:       gateID,
					RuleID:     "no-duplicate-function",
					Severity:   finding.SeverityWarn,
					File:       f.Path,
					Line:       fn.StartLine,
					Message:    fmt.Sprintf("function %s is %.0f%% similar to %s in %s", fn.Name, score*100, match.Name, match.File),
					Suggestion: fmt.Sprintf("reuse or extract the existing %s in %s instead of duplicating it", match.Name, match.File),
					RetryHint:  fmt.Sprintf("Function %s duplicates existing %s (%s). Reuse the existing implementation instead of adding a near-copy.", fn.Name, match.Name, match.File),
				})
			}
		}
	}
	return findings, nil
}

// touchesAdded reports whether the function's line range includes an added line.
func touchesAdded(fn repoindex.FuncInfo, added map[int]bool) bool {
	for ln := fn.StartLine; ln <= fn.EndLine; ln++ {
		if added[ln] {
			return true
		}
	}
	return false
}

// bestMatch returns the most similar existing function that is not the same
// function (by file+name), but only when its score clears the threshold.
func bestMatch(fn repoindex.FuncInfo, index *repoindex.Index, selfFile string, threshold float64) (*repoindex.FuncInfo, float64) {
	var best *repoindex.FuncInfo
	var bestScore float64
	for i := range index.Funcs {
		cand := &index.Funcs[i]
		if cand.File == selfFile && cand.Name == fn.Name {
			continue // the function itself
		}
		score := repoindex.Jaccard(fn.Shingles, cand.Shingles)
		if score > bestScore {
			best, bestScore = cand, score
		}
	}
	if best == nil || bestScore < threshold {
		return nil, 0
	}
	return best, bestScore
}
