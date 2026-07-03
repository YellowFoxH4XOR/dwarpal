// Package archrules implements user-defined architecture rules (PRD §5.3).
//
// A rule names a regexp over call-expression text (e.g. "sql.Open|db.Query")
// and a set of ForbiddenOutside globs. The polarity is inverted from a plain
// allow-list: matching calls are permitted only inside paths matching one of
// those globs (e.g. the repo layer) and forbidden everywhere else (e.g. web
// handlers reaching straight into the database instead of going through the
// repo layer). This mirrors the PRD's canonical example.
//
// Languages: Go via go/parser (ast.CallExpr), and Python/TypeScript/JavaScript
// via the tree-sitter astengine (call-expression queries). A rule declares its
// Language; a rule targeting an unsupported language is a loud config error
// (fail closed), not a silent no-op — a layering rule you think is enforced but
// isn't is worse than no rule.
package archrules

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YellowFoxH4XOR/dwarpal/internal/astengine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "architecture_rules"

// supportedLangs are the languages architecture rules can be enforced in.
var supportedLangs = map[string]bool{"go": true, "python": true, "typescript": true, "javascript": true}

// callQueries capture the callee of a call expression; the capture's source
// text (e.g. "db.query", "sql.Open") is the rendered target the rule matches.
var callQueries = map[astengine.Language]string{
	astengine.LangTS: `(call_expression function: (_) @callee)`,
	astengine.LangJS: `(call_expression function: (_) @callee)`,
	astengine.LangPy: `(call function: (_) @callee)`,
}

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

type compiledRule struct {
	rule Rule
	re   *regexp.Regexp
}

// callSite is one call expression's rendered target and 1-indexed line.
type callSite struct {
	text string
	line int
}

// Gate enforces the configured architecture rules against a diff.
type Gate struct {
	root  string
	rules []Rule
}

// New builds the architecture-rules gate. root is the repo root the changed
// paths are resolved against.
func New(root string, rules []Rule) *Gate { return &Gate{root: root, rules: rules} }

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// Run flags added calls that match a rule's Matches regexp from a path outside
// that rule's ForbiddenOutside globs, per language. Invalid regexps and rules
// for unsupported languages are configuration errors (fail closed) surfaced as
// GateErrors, never silently skipped.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	byLang := map[string][]compiledRule{}
	for _, r := range g.rules {
		if !supportedLangs[r.Language] {
			return nil, fmt.Errorf("architecture rule %s: unsupported language %q (supported: go, python, typescript, javascript)", r.ID, r.Language)
		}
		re, err := regexp.Compile(r.Matches)
		if err != nil {
			return nil, fmt.Errorf("architecture rule %s: invalid matches regexp: %w", r.ID, err)
		}
		byLang[r.Language] = append(byLang[r.Language], compiledRule{rule: r, re: re})
	}
	if len(byLang) == 0 {
		return nil, nil
	}

	var findings []finding.Finding
	for _, f := range d.Files {
		if len(f.AddedLines) == 0 {
			continue
		}
		rules := byLang[fileLang(f.Path)]
		if len(rules) == 0 {
			continue
		}
		added := make(map[int]bool, len(f.AddedLines))
		for _, ln := range f.AddedLines {
			added[ln.Number] = true
		}
		for _, cs := range g.callSites(f.Path) {
			if !added[cs.line] {
				continue
			}
			for _, cr := range rules {
				if !cr.re.MatchString(cs.text) || g.allowedHere(f.Path, cr.rule.ForbiddenOutside) {
					continue
				}
				findings = append(findings, finding.Finding{
					Gate:      gateID,
					RuleID:    cr.rule.ID,
					Severity:  severityOf(cr.rule.Severity),
					File:      f.Path,
					Line:      cs.line,
					Message:   cr.rule.Description,
					RetryHint: fmt.Sprintf("%s is only allowed in paths matching %v. Move this call behind that layer instead of calling it directly from %s.", cs.text, cr.rule.ForbiddenOutside, f.Path),
				})
			}
		}
	}
	return findings, nil
}

// fileLang maps a path to a rule Language value, or "" if unsupported.
func fileLang(path string) string {
	if strings.HasSuffix(path, ".go") {
		return "go"
	}
	switch astengine.LanguageFor(path) {
	case astengine.LangTS:
		return "typescript"
	case astengine.LangJS:
		return "javascript"
	case astengine.LangPy:
		return "python"
	}
	return ""
}

// callSites extracts every call expression's rendered target + line from a
// changed file, dispatching to the Go parser or the tree-sitter engine.
// Unparsable source yields no sites (nothing we can check), never an error.
func (g *Gate) callSites(path string) []callSite {
	abs := filepath.Join(g.root, path)
	if strings.HasSuffix(path, ".go") {
		return goCallSites(abs)
	}
	return tsCallSites(abs, path)
}

func goCallSites(abs string) []callSite {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, abs, nil, 0)
	if err != nil {
		return nil
	}
	var sites []callSite
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if text := callText(call.Fun); text != "" {
				sites = append(sites, callSite{text: text, line: fset.Position(call.Pos()).Line})
			}
		}
		return true
	})
	return sites
}

func tsCallSites(abs, path string) []callSite {
	lang := astengine.LanguageFor(path)
	query, ok := callQueries[lang]
	if !ok {
		return nil
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	tree, err := astengine.Parse(path, src)
	if err != nil {
		return nil // parse failure: nothing we can check, not an error
	}
	caps, err := tree.Query(query)
	if err != nil {
		return nil
	}
	var sites []callSite
	for _, c := range caps {
		if c.Name == "callee" && c.Text != "" {
			sites = append(sites, callSite{text: c.Text, line: c.StartLine})
		}
	}
	return sites
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

// callText renders a Go call target as "pkg.Fn" / "recv.Method" text for
// regexp matching. Only selector and bare-identifier targets render; anything
// else (e.g. func literals) yields "".
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
