// Package audit self-calibrates Dwarpal's rules against the repo's own git
// history. For each ai_patterns rule it measures the "acted-on rate": of the
// lines the rule flagged in recent commits, how many did a human later rewrite
// or remove. A rule people act on catches real problems (signal); a rule whose
// flags survive untouched is noise. This is the BitsAI-CR "outdated rate"
// (arXiv 2501.15134), computed from git history alone — no LLM, no network, no
// telemetry — which is exactly why a cloud reviewer can't replicate it on a
// local repo.
//
// Like analyze, audit is deterministic, offline, and print-only: it never
// mutates .dwarpal.yml. The agent or a human decides what to do with the
// signal.
package audit

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gates/aipatterns"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// Options tunes an audit run.
type Options struct {
	Window           int     // most-recent non-merge commits to replay
	MinSamples       int     // below this, a rule has too little signal to advise
	DemoteThreshold  float64 // acted-on rate at/below → recommend demote to warn
	PromoteThreshold float64 // acted-on rate at/above → suggest promote (review)
}

// Defaults returns the documented default thresholds.
func Defaults() Options {
	return Options{Window: 200, MinSamples: 8, DemoteThreshold: 0.15, PromoteThreshold: 0.6}
}

// Report is the full calibration result, JSON-serializable for agents.
type Report struct {
	Window         int        `json:"window"`
	CommitsScanned int        `json:"commits_scanned"`
	Rules          []RuleStat `json:"rules"`
}

// RuleStat is one rule's calibration.
type RuleStat struct {
	Gate            string  `json:"gate"`
	RuleID          string  `json:"rule_id"`
	CurrentSeverity string  `json:"current_severity"`
	Samples         int     `json:"samples"`
	ActedOn         int     `json:"acted_on"`
	ActedOnRate     float64 `json:"acted_on_rate"`
	Recommendation  string  `json:"recommendation,omitempty"`
}

// flag is one rule firing on one added line in one historical commit — the unit
// whose later fate we resolve against HEAD.
type flag struct {
	gate, ruleID, severity string
	file, lineText         string
}

// Run replays the last opts.Window non-merge commits through ai_patterns and
// reports each rule's acted-on rate. Purely local: it reads git history and
// materializes historical blobs into a throwaway scratch dir, nothing else.
func Run(root string, opts Options) (*Report, error) {
	commits, err := recentCommits(root, opts.Window)
	if err != nil {
		return nil, err
	}

	ext := gitio.NewExtractor(root)
	scratch, err := os.MkdirTemp("", "dwarpal-audit-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(scratch)

	var flags []flag
	scanned := 0
	for _, c := range commits {
		fs, ok := replayCommit(root, ext, c, scratch)
		if !ok {
			continue // root commit (no parent) or unreadable — skip, don't fail
		}
		scanned++
		flags = append(flags, fs...)
	}

	rep := &Report{Window: opts.Window, CommitsScanned: scanned}
	rep.Rules = aggregate(root, flags, opts)
	return rep, nil
}

// replayCommit runs ai_patterns against commit c's diff, with the AST tier
// pointed at c's version of the touched files (materialized into scratch). It
// returns one flag per finding, carrying the flagged line's text for later
// resolution. ok=false means the commit has no parent or couldn't be read.
func replayCommit(root string, ext *gitio.Extractor, c, scratch string) ([]flag, bool) {
	d, err := ext.Range(c + "^.." + c)
	if err != nil || d.Empty() {
		return nil, false
	}

	// Materialize c's version of each touched file so the AST tier parses the
	// historical content, not the current work tree.
	commitDir := filepath.Join(scratch, c)
	if err := materialize(root, c, d, commitDir); err != nil {
		return nil, false
	}
	defer os.RemoveAll(commitDir)

	findings, err := aipatterns.New(commitDir, nil).Run(context.Background(), d, engine.NoIndex{})
	if err != nil {
		return nil, false
	}

	text := addedLineText(d)
	flags := make([]flag, 0, len(findings))
	for _, f := range findings {
		flags = append(flags, flag{
			gate:     f.Gate,
			ruleID:   f.RuleID,
			severity: string(f.Severity),
			file:     f.File,
			lineText: strings.TrimSpace(text[lineKey(f.File, f.Line)]),
		})
	}
	return flags, true
}

// aggregate resolves every flag against HEAD and rolls the results up per rule.
func aggregate(root string, flags []flag, opts Options) []RuleStat {
	type acc struct {
		severity     string
		samples, hit int
	}
	byRule := map[string]*acc{}
	headCache := map[string]*string{} // path → HEAD content (nil = absent)

	for _, fl := range flags {
		if fl.lineText == "" {
			continue // no text to resolve against (e.g. AST finding without a captured line)
		}
		key := fl.gate + "/" + fl.ruleID
		a := byRule[key]
		if a == nil {
			a = &acc{severity: fl.severity}
			byRule[key] = a
		}
		a.samples++
		if actedOn(root, fl, headCache) {
			a.hit++
		}
	}

	stats := make([]RuleStat, 0, len(byRule))
	for key, a := range byRule {
		parts := strings.SplitN(key, "/", 2)
		rate := 0.0
		if a.samples > 0 {
			rate = float64(a.hit) / float64(a.samples)
		}
		stats = append(stats, RuleStat{
			Gate:            parts[0],
			RuleID:          parts[1],
			CurrentSeverity: a.severity,
			Samples:         a.samples,
			ActedOn:         a.hit,
			ActedOnRate:     round2(rate),
			Recommendation:  recommend(rate, a.samples, a.severity, opts),
		})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Samples != stats[j].Samples {
			return stats[i].Samples > stats[j].Samples
		}
		return stats[i].Gate+stats[i].RuleID < stats[j].Gate+stats[j].RuleID
	})
	return stats
}

// actedOn reports whether the flagged line was later rewritten or removed by
// HEAD: the file is gone, or its exact flagged line text no longer appears.
func actedOn(root string, fl flag, cache map[string]*string) bool {
	content, seen := cache[fl.file]
	if !seen {
		content = headBlob(root, fl.file)
		cache[fl.file] = content
	}
	if content == nil {
		return true // file removed (or renamed away) — treated as acted-on
	}
	return !strings.Contains(*content, fl.lineText)
}

// recommend turns an acted-on rate into advice. It only ever *suggests*
// promotion (review manually) — v1 never auto-promotes a rule to hard-block on
// this fuzzy signal, which is the design's worst failure mode.
func recommend(rate float64, samples int, severity string, opts Options) string {
	if samples < opts.MinSamples {
		return "" // too little signal
	}
	if rate <= opts.DemoteThreshold && severity == "error" {
		return "demote to warn (rarely acted on)"
	}
	if rate >= opts.PromoteThreshold && severity != "error" {
		return "review for promotion to error (often acted on)"
	}
	return ""
}
