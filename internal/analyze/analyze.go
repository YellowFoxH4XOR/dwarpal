// Package analyze measures a repository and produces facts an agent uses to
// author a .dwarpal.yml consistent with the codebase.
//
// Design: Dwarpal itself never calls an LLM. The developer is already inside a
// coding agent (Claude Code, Codex, ...) with the whole repo in context — that
// agent is the config author. analyze just gives it deterministic, offline
// facts (and the config format, via the agent-setup instructions) so the
// generated config reflects how the repo actually works, not a generic guess.
package analyze

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/repoindex"
)

// Report is the full set of measured facts, JSON-serializable for agents.
type Report struct {
	Languages        []string            `json:"languages"`
	DiffBudget       BudgetSuggestion    `json:"diff_budget"`
	Conventions      map[string]LangConv `json:"conventions"`
	CoverageArtifact string              `json:"coverage_artifact,omitempty"`
	SecurityTools    []string            `json:"security_tools,omitempty"`
	BranchPrefixes   []string            `json:"branch_prefixes,omitempty"`
	LayeringHints    []string            `json:"layering_hints,omitempty"`
}

// BudgetSuggestion is a diff budget fitted to the repo's own history, not a
// blind default — a repo that commits in 80-line chunks shouldn't inherit 500.
type BudgetSuggestion struct {
	MaxLines    int    `json:"max_lines"`
	Basis       string `json:"basis"`
	SampleCount int    `json:"commits_sampled"`
	// The distribution, so the agent can judge rather than trust one number.
	MedianLines int `json:"median_commit_lines,omitempty"`
	P75Lines    int `json:"p75_commit_lines,omitempty"`
	P90Lines    int `json:"p90_commit_lines,omitempty"`
	MaxSeen     int `json:"max_commit_lines,omitempty"`
}

// LangConv is one language's dominant conventions.
type LangConv struct {
	DominantImportForm string  `json:"dominant_import_form,omitempty"`
	ImportShare        float64 `json:"import_share,omitempty"`
	DominantErrorIdiom string  `json:"dominant_error_idiom,omitempty"`
	ErrorIdiomShare    float64 `json:"error_idiom_share,omitempty"`
	DominantNaming     string  `json:"dominant_naming,omitempty"` // snake_case | camelCase
	SnakeCaseFuncs     int     `json:"snake_case_funcs,omitempty"`
	Funcs              int     `json:"funcs,omitempty"`
	AvgFuncLines       int     `json:"avg_func_lines,omitempty"`
}

// Run analyzes root and returns the measured facts. Purely local: it reads the
// work tree and git history, never the network.
func Run(root string) (*Report, error) {
	r := &Report{Conventions: map[string]LangConv{}}

	idx, err := repoindex.BuildFor(root, true)
	if err == nil {
		r.fromIndex(idx)
	}
	r.DiffBudget = suggestBudget(root)
	r.CoverageArtifact = detectCoverage(root)
	r.SecurityTools = detectTools(root)
	r.BranchPrefixes = detectBranchPrefixes(root)
	r.LayeringHints = detectLayering(root)
	return r, nil
}

// fromIndex translates the repo fingerprint into per-language conventions and
// the language list.
func (r *Report) fromIndex(idx *repoindex.Index) {
	c := idx.Conventions
	seen := map[string]bool{}

	// Per-language function conventions (naming case, size) — now for every
	// language, not just Go, via the per-language FuncByLang counts.
	for lang, s := range c.FuncByLang {
		if s.Funcs == 0 {
			continue
		}
		lc := r.Conventions[lang]
		lc.Funcs = s.Funcs
		lc.SnakeCaseFuncs = s.SnakeCaseFuncs
		lc.AvgFuncLines = int(c.AvgFuncLinesFor(lang) + 0.5)
		if ratio, _ := c.SnakeRatio(lang); ratio >= 0.85 {
			lc.DominantNaming = "snake_case"
		} else if ratio <= 0.15 {
			lc.DominantNaming = "camelCase"
		}
		r.Conventions[lang] = lc
		seen[lang] = true
	}

	// Error idiom is a Go-specific concept.
	if idiom, share := c.DominantErrorIdiom(); idiom != "" {
		lc := r.Conventions["go"]
		lc.DominantErrorIdiom, lc.ErrorIdiomShare = idiom, round2(share)
		r.Conventions["go"] = lc
		seen["go"] = true
	}

	// Import style per language.
	for lang := range c.Imports {
		if lang == "go-error-idiom" {
			continue
		}
		lc := r.Conventions[lang]
		if form, share := c.DominantImportForm(lang); form != "" {
			lc.DominantImportForm, lc.ImportShare = form, round2(share)
		}
		r.Conventions[lang] = lc
		seen[lang] = true
	}

	for lang := range seen {
		r.Languages = append(r.Languages, lang)
	}
	sort.Strings(r.Languages)
	r.Languages = dedupe(r.Languages)
}

// suggestBudget fits max_lines to the repo's recent commit-size distribution:
// the ~90th percentile of non-merge commit change-counts, so normal commits
// pass and only genuine outliers trip the gate. Falls back to the PRD default.
func suggestBudget(root string) BudgetSuggestion {
	out, err := gitOut(root, "log", "--no-merges", "-n", "200", "--format=%H", "--shortstat")
	if err != nil {
		return BudgetSuggestion{MaxLines: 500, Basis: "default (no history)", SampleCount: 0}
	}
	var sizes []int
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "insertion") && !strings.Contains(line, "deletion") {
			continue
		}
		sizes = append(sizes, parseShortstat(line))
	}
	if len(sizes) < 10 {
		return BudgetSuggestion{MaxLines: 500, Basis: "default (too little history)", SampleCount: len(sizes)}
	}
	sort.Ints(sizes)
	pct := func(p int) int { return sizes[min(len(sizes)*p/100, len(sizes)-1)] }
	// Base the budget on the 75th percentile, not the 90th/max: bootstrap and
	// generated-code commits are real but shouldn't set everyday policy. 2x
	// headroom over p75 lets normal work through while flagging genuine dumps.
	budget := roundUp(int(float64(pct(75))*2.0), 50)
	if budget < 100 {
		budget = 100
	}
	return BudgetSuggestion{
		MaxLines: budget, SampleCount: len(sizes),
		Basis:       "2x the 75th-percentile commit size (outlier-robust)",
		MedianLines: pct(50), P75Lines: pct(75), P90Lines: pct(90), MaxSeen: sizes[len(sizes)-1],
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseShortstat pulls the total changed lines from a git --shortstat line
// like " 3 files changed, 42 insertions(+), 7 deletions(-)".
func parseShortstat(line string) int {
	total := 0
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		if strings.HasPrefix(fields[1], "insertion") || strings.HasPrefix(fields[1], "deletion") {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				total += n
			}
		}
	}
	return total
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func roundUp(n, to int) int {
	if to == 0 {
		return n
	}
	return ((n + to - 1) / to) * to
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func gitOut(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	return string(out), err
}
