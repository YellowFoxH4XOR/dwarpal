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

	"github.com/YellowFoxH4XOR/dwarpal/internal/astengine"
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

	var findings []finding.Finding
	for _, f := range d.Files {
		// Import-style dimension: any supported language.
		findings = append(findings, g.importStyleFindings(conv, f)...)
		// Error-idiom dimension (#37): Go only — it's a Go-specific concept.
		if strings.HasSuffix(f.Path, ".go") {
			findings = append(findings, g.errorIdiomFindings(conv, f)...)
		}
		// Naming + size dimensions: per language, against that language's norm.
		findings = append(findings, g.functionFindings(conv, f)...)
	}
	return findings, nil
}

// functionFindings scores added functions against the repo's convention norm
// FOR THEIR LANGUAGE: naming case that bucks a strong majority, and functions
// far longer than that language's typical size. It skips a language without a
// large enough sample to have a meaningful norm, so a repo's Python files are
// judged by Python conventions and its Go files by Go conventions.
func (g *Gate) functionFindings(conv repoindex.Conventions, f gitio.FileChange) []finding.Finding {
	lang := repoindex.LangLabel(f.Path)
	if lang == "" || len(f.AddedLines) == 0 {
		return nil
	}
	snakeRatio, n := conv.SnakeRatio(lang)
	if n < 5 {
		return nil // too little of this language to have a naming/size norm
	}
	ex := repoindex.FunctionsFor(f.Path)
	if ex == nil {
		return nil
	}
	src, err := os.ReadFile(filepath.Join(g.root, f.Path))
	if err != nil {
		return nil
	}
	added := make(map[int]bool, len(f.AddedLines))
	for _, ln := range f.AddedLines {
		added[ln.Number] = true
	}
	avg := conv.AvgFuncLinesFor(lang)

	var out []finding.Finding
	for _, fn := range ex(f.Path, src) {
		if !touches(fn, added) {
			continue
		}
		if rule, msg := namingDrift(lang, snakeRatio, fn.Name); rule != "" {
			out = append(out, g.finding(f.Path, fn.StartLine, rule, msg,
				"rename to match the repo's "+lang+" naming convention"))
		}
		length := fn.EndLine - fn.StartLine + 1
		if avg > 0 && float64(length) > 3*avg {
			out = append(out, g.finding(f.Path, fn.StartLine, "function-size",
				fmt.Sprintf("function %s is %d lines; the repo's %s average is %.0f", fn.Name, length, lang, avg),
				"consider splitting this function to match the repo's typical size"))
		}
	}
	return out
}

// namingDrift flags a naming-convention outlier against the language's strong
// majority: a snake_case name in an overwhelmingly camelCase language (Go, JS),
// or a camelCase name in an overwhelmingly snake_case one (Python). A mixed repo
// (no strong majority either way) produces nothing — the thresholds leave a
// deliberate dead zone so drift only fires on a clear norm.
func namingDrift(lang string, snakeRatio float64, name string) (rule, msg string) {
	switch {
	case snakeRatio <= 0.15 && strings.Contains(name, "_"):
		return "naming-style", fmt.Sprintf("function %s uses snake_case; the repo's %s is overwhelmingly camelCase", name, lang)
	case snakeRatio >= 0.85 && strings.ToLower(name) != name:
		return "naming-style", fmt.Sprintf("function %s uses camelCase; the repo's %s is overwhelmingly snake_case", name, lang)
	}
	return "", ""
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

// importStyleFindings flags added import lines whose form disagrees with a
// strong repo majority (dominant form >= 80%). Info severity: an import style
// is a convention, not a defect.
func (g *Gate) importStyleFindings(conv repoindex.Conventions, f gitio.FileChange) []finding.Finding {
	lang := astengine.LanguageFor(f.Path)
	if lang == "" {
		return nil
	}
	dominant, share := conv.DominantImportForm(string(lang))
	if dominant == "" || share < 0.8 {
		return nil // no strong norm to drift from
	}
	var out []finding.Finding
	for _, ln := range f.AddedLines {
		form := repoindex.ClassifyImportLine(lang, ln.Text)
		if form == "" || form == dominant {
			continue
		}
		out = append(out, g.finding(f.Path, ln.Number,
			"import-style",
			fmt.Sprintf("%s import in a repo where %.0f%% of imports are %s", form, share*100, dominant),
			fmt.Sprintf("use the repo's dominant %s import form", dominant)))
	}
	return out
}

// errorIdiomFindings flags added Go error-handling lines whose idiom disagrees
// with a strong repo majority (>= 80%). A repo that consistently wraps errors
// gets told about a bare `return err`; a panic-free repo gets told about a new
// panic. Info severity — idioms are conventions.
func (g *Gate) errorIdiomFindings(conv repoindex.Conventions, f gitio.FileChange) []finding.Finding {
	dominant, share := conv.DominantErrorIdiom()
	if dominant == "" || share < 0.8 {
		return nil
	}
	var out []finding.Finding
	for _, ln := range f.AddedLines {
		idiom := repoindex.ClassifyErrorIdiomLine(ln.Text)
		if idiom == "" || idiom == dominant {
			continue
		}
		out = append(out, g.finding(f.Path, ln.Number,
			"error-idiom",
			fmt.Sprintf("%s error handling in a repo where %.0f%% of error handling uses %s", idiom, share*100, dominant),
			fmt.Sprintf("follow the repo's dominant %s idiom", dominant)))
	}
	return out
}

func touches(fn repoindex.FuncInfo, added map[int]bool) bool {
	for ln := fn.StartLine; ln <= fn.EndLine; ln++ {
		if added[ln] {
			return true
		}
	}
	return false
}
