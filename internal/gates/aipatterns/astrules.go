package aipatterns

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/astengine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// AST-precise tier of no-broad-catch / no-sql-concat for TS/JS/Python
// (tree-sitter-ast-engine change, design D5). Queries anchor findings to real
// catch/except-clause and string-expression nodes; small verdict functions on
// the captured text decide compliance. Files this tier handles are excluded
// from the regex heuristics for these two rules (no double reporting) — the
// regex tier continues to serve every other language.

// catchQueries capture the handler body of catch/except constructs.
var catchQueries = map[astengine.Language]string{
	astengine.LangTS: `(catch_clause body: (statement_block) @body) @clause`,
	astengine.LangJS: `(catch_clause body: (statement_block) @body) @clause`,
	astengine.LangPy: `(except_clause) @clause`,
}

// sqlStringQueries capture string-ish expressions that can smuggle SQL.
var sqlStringQueries = map[astengine.Language]string{
	astengine.LangTS: `[(template_string) @s (binary_expression (string) @s)]`,
	astengine.LangJS: `[(template_string) @s (binary_expression (string) @s)]`,
	astengine.LangPy: `[(string) @s (binary_operator (string) @s)]`,
}

var sqlKeyword = regexp.MustCompile(`(?i)\b(select\s|insert\s+into|update\s+\w+\s+set|delete\s+from)`)

// astRuleFindings runs the AST-precise tier for one changed file. handled
// reports whether this tier covered the file (so the regex heuristics for
// these rules are suppressed for it).
func astRuleFindings(root string, f gitio.FileChange) (findings []finding.Finding, handled bool) {
	lang := astengine.LanguageFor(f.Path)
	if lang == "" || len(f.AddedLines) == 0 {
		return nil, false
	}
	src, err := os.ReadFile(filepath.Join(root, f.Path))
	if err != nil {
		return nil, false
	}
	tree, err := astengine.Parse(f.Path, src)
	if err != nil {
		return nil, false // full degradation: regex tier serves this file
	}

	added := map[int]bool{}
	for _, ln := range f.AddedLines {
		added[ln.Number] = true
	}

	findings = append(findings, broadCatchFindings(tree, lang, f.Path, added)...)
	findings = append(findings, sqlConcatFindings(tree, lang, f.Path, added)...)
	return findings, true
}

// broadCatchFindings flags catch/except handlers on added lines that swallow
// the error: an empty body, or one that neither re-raises/throws nor calls
// anything (the "log or rethrow" test — any call is treated as handling).
func broadCatchFindings(tree *astengine.Tree, lang astengine.Language, path string, added map[int]bool) []finding.Finding {
	caps, err := tree.Query(catchQueries[lang])
	if err != nil {
		return nil
	}
	var out []finding.Finding
	for _, c := range caps {
		if c.Name != "clause" && c.Name != "body" {
			continue
		}
		// For TS/JS the @body capture is the statement block; for Python the
		// @clause capture includes the handler suite.
		if lang != astengine.LangPy && c.Name != "body" {
			continue
		}
		if lang == astengine.LangPy && c.Name != "clause" {
			continue
		}
		if !touchesAddedLines(c.StartLine, c.EndLine, added) {
			continue
		}
		if handlerSwallows(lang, c.Text) {
			out = append(out, finding.Finding{
				Gate:       gateID,
				RuleID:     "no-broad-catch",
				Severity:   finding.SeverityWarn,
				File:       path,
				Line:       c.StartLine,
				Message:    "exception handler swallows the error (no rethrow, no call)",
				Suggestion: "narrow the catch and log or rethrow instead of swallowing",
				RetryHint:  "This handler silently swallows errors. Re-raise, or handle it with an explicit call (e.g. logging).",
			})
		}
	}
	return out
}

// handlerSwallows decides whether a handler body swallows the error.
func handlerSwallows(lang astengine.Language, body string) bool {
	if lang == astengine.LangPy {
		// Strip the clause header ("except ...:"); look at the suite.
		if i := strings.Index(body, ":"); i >= 0 {
			body = body[i+1:]
		}
		trimmed := strings.TrimSpace(body)
		if trimmed == "pass" || trimmed == "" || trimmed == "..." {
			return true
		}
		return !strings.Contains(body, "raise") && !strings.Contains(body, "(")
	}
	inner := strings.TrimSpace(strings.Trim(strings.TrimSpace(body), "{}"))
	if inner == "" {
		return true
	}
	return !strings.Contains(inner, "throw") && !strings.Contains(inner, "(")
}

// sqlConcatFindings flags string expressions on added lines that build SQL by
// interpolation (`${...}` / f-string braces) or that sit inside a `+`
// concatenation, per the diff-local v1 contract.
func sqlConcatFindings(tree *astengine.Tree, lang astengine.Language, path string, added map[int]bool) []finding.Finding {
	caps, err := tree.Query(sqlStringQueries[lang])
	if err != nil {
		return nil
	}
	var out []finding.Finding
	seenLine := map[int]bool{}
	for _, c := range caps {
		if !touchesAddedLines(c.StartLine, c.EndLine, added) || seenLine[c.StartLine] {
			continue
		}
		if !sqlKeyword.MatchString(c.Text) {
			continue
		}
		if !stringBuildsSQL(lang, c.Text) {
			continue
		}
		seenLine[c.StartLine] = true
		out = append(out, finding.Finding{
			Gate:       gateID,
			RuleID:     "no-sql-concat",
			Severity:   finding.SeverityWarn,
			File:       path,
			Line:       c.StartLine,
			Message:    "SQL built by string interpolation/concatenation",
			Suggestion: "use parameterized queries instead of interpolating values into SQL",
			RetryHint:  "Rewrite this SQL to use bound parameters rather than interpolation or concatenation.",
		})
	}
	return out
}

// stringBuildsSQL reports whether the captured string actively interpolates.
// A plain SQL string literal (no interpolation) inside a binary_expression is
// caught by the query shape itself; template/f-strings need visible splices.
func stringBuildsSQL(lang astengine.Language, text string) bool {
	if lang == astengine.LangPy {
		lower := strings.ToLower(text)
		isFString := strings.HasPrefix(lower, `f"`) || strings.HasPrefix(lower, "f'")
		return (isFString && strings.Contains(text, "{")) || !isFString
	}
	if strings.HasPrefix(text, "`") {
		return strings.Contains(text, "${")
	}
	return true // plain string captured inside a + expression
}

func touchesAddedLines(start, end int, added map[int]bool) bool {
	for ln := start; ln <= end; ln++ {
		if added[ln] {
			return true
		}
	}
	return false
}
