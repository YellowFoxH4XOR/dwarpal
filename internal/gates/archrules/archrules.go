// Package archrules implements user-defined architecture rules (PRD §5.3,
// task #47 — the biggest in-spec gap at the time this gate was written).
//
// A rule names a regexp over call-expression text (e.g. "sql.Open|db.Query")
// and a set of ForbiddenOutside globs. The polarity is inverted from a plain
// allow-list: matching calls are permitted only inside paths matching one of
// those globs (e.g. the repo layer) and forbidden everywhere else (e.g. web
// handlers reaching straight into the database instead of going through the
// repo layer). This mirrors the PRD's canonical example.
//
// v1 is Go-only: it parses each changed .go file from disk with go/parser and
// walks ast.CallExpr nodes, so it needs no build/type info. Rules for other
// languages are accepted but skipped silently until a grammar lands for them.
package archrules

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "architecture_rules"

// Rule is one user-defined architecture constraint. Matches is a regexp over
// the rendered call-expression text (e.g. "pkg.Fn" or "recv.Method").
// ForbiddenOutside lists globs where matching calls ARE allowed; everywhere
// else they are forbidden. Severity defaults to error when empty.
type Rule struct {
	ID               string
	Description      string
	Language         string
	Matches          string
	ForbiddenOutside []string
	Severity         string
}

// compiledRule pairs a Rule with its parsed regexp so we compile once per Run.
type compiledRule struct {
	rule Rule
	re   *regexp.Regexp
}

// Gate enforces the configured architecture rules against a diff.
type Gate struct {
	root  string
	rules []Rule
}

// New builds the architecture-rules gate. root is the repo root the changed
// paths are resolved against.
func New(root string, rules []Rule) *Gate {
	return &Gate{root: root, rules: rules}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run parses each changed Go file and flags added calls that match a rule's
// Matches regexp from a path outside that rule's ForbiddenOutside globs.
//
// Invalid Matches regexps are a configuration error, not a code problem — we
// return an error (fail closed) so the caller surfaces it as a GateError
// rather than silently skipping the rule.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	compiled := make([]compiledRule, 0, len(g.rules))
	for _, r := range g.rules {
		if r.Language != "go" {
			continue // other languages: no grammar yet, skip silently
		}
		re, err := regexp.Compile(r.Matches)
		if err != nil {
			return nil, fmt.Errorf("architecture rule %s: invalid matches regexp: %w", r.ID, err)
		}
		compiled = append(compiled, compiledRule{rule: r, re: re})
	}
	if len(compiled) == 0 {
		return nil, nil
	}

	var findings []finding.Finding
	for _, f := range d.Files {
		if filepath.Ext(f.Path) != ".go" || len(f.AddedLines) == 0 {
			continue
		}
		added := map[int]bool{}
		for _, ln := range f.AddedLines {
			added[ln.Number] = true
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(g.root, f.Path), nil, 0)
		if err != nil {
			continue // unparsable source (e.g. build tag oddities) — nothing we can check
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			text := callText(call.Fun)
			if text == "" {
				return true
			}
			line := fset.Position(call.Pos()).Line
			if !added[line] {
				return true
			}
			for _, cr := range compiled {
				if !cr.re.MatchString(text) {
					continue
				}
				if g.allowedHere(f.Path, cr.rule.ForbiddenOutside) {
					continue
				}
				findings = append(findings, finding.Finding{
					Gate:      gateID,
					RuleID:    cr.rule.ID,
					Severity:  severityOf(cr.rule.Severity),
					File:      f.Path,
					Line:      line,
					Message:   cr.rule.Description,
					RetryHint: fmt.Sprintf("%s is only allowed in paths matching %v. Move this call behind that layer instead of calling it directly from %s.", text, cr.rule.ForbiddenOutside, f.Path),
				})
			}
			return true
		})
	}
	return findings, nil
}

// allowedHere reports whether path matches one of the rule's ForbiddenOutside
// globs — i.e. this is a location where the matched call is permitted.
func (g *Gate) allowedHere(path string, globs []string) bool {
	for _, glob := range globs {
		if ok, _ := doublestar.Match(glob, path); ok {
			return true
		}
	}
	return false
}

// severityOf maps a rule's configured severity string to finding.Severity,
// defaulting to error so a misconfigured (empty) severity fails closed.
func severityOf(s string) finding.Severity {
	switch s {
	case "warn":
		return finding.SeverityWarn
	case "info":
		return finding.SeverityInfo
	default:
		return finding.SeverityError
	}
}

// callText renders a call target as "pkg.Fn" / "recv.Method" text for
// regexp matching. Only selector (pkg.Fn / x.Fn) and bare identifier (Fn)
// call targets are rendered; anything else (e.g. func literals) yields "".
func callText(fun ast.Expr) string {
	switch e := fun.(type) {
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok {
			return ident.Name + "." + e.Sel.Name
		}
		return callText(e.X) + "." + e.Sel.Name
	case *ast.Ident:
		return e.Name
	default:
		return ""
	}
}
