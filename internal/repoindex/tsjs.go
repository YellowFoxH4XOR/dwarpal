// Heuristic TypeScript/JavaScript function extraction. There is no bundled
// TS/JS parser in the Go standard library and dwarpal avoids CGO/tree-sitter
// (see repoindex.go's package doc for the v1 tradeoff), so this file finds
// function-shaped constructs with regexes and then locates their body by
// balanced-brace scanning. This covers top-level function declarations,
// `const x = (...) => {...}` arrow functions, and class methods — the shapes
// the duplicate-function gate needs to compare. Anonymous callbacks nested
// inside other functions are not separately extracted; they are covered
// implicitly as part of their enclosing function's body/shingles.
package repoindex

import (
	"regexp"
	"strings"
)

// tsFuncHeader matches the start of a function-shaped declaration:
//   - function name(...)
//   - const/let/var name = (...) =>   or   name = async (...) =>
//   - name(...) {              (class/object method shorthand)
//
// It intentionally does not try to fully parse parameter lists; it only
// needs to find the name and the position to start brace-balancing from.
var tsFuncHeader = regexp.MustCompile(
	`(?m)^[ \t]*(?:export[ \t]+)?(?:default[ \t]+)?(?:async[ \t]+)?` +
		`(?:function\*?[ \t]+(?P<fn>[A-Za-z_$][\w$]*)[ \t]*\(|` +
		`(?:const|let|var)[ \t]+(?P<arrow>[A-Za-z_$][\w$]*)[ \t]*=[ \t]*(?:async[ \t]*)?\([^\n]*\)[ \t]*(?::[^=\n{]*)?=>[ \t]*\{|` +
		`(?P<method>[A-Za-z_$][\w$]*)[ \t]*\([^\n]*\)[ \t]*\{)`,
)

// tsControlKeywords are identifiers that look like the "method" alternative
// (`name(...) {`) but are actually control-flow statements, not functions.
var tsControlKeywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "catch": true,
	"function": true, "return": true, "else": true, "try": true, "do": true,
}

// FunctionsInTSSource heuristically extracts top-level and method-level
// functions from a TypeScript/JavaScript source blob. StartLine is the line
// of the header match; EndLine is the line of the matching closing brace
// found by balanced-brace scanning from the header's opening brace.
func FunctionsInTSSource(rel string, src []byte) []FuncInfo {
	var out []FuncInfo
	matches := tsFuncHeader.FindAllSubmatchIndex(src, -1)
	names := tsFuncHeader.SubexpNames()

	for _, m := range matches {
		var name string
		for i, n := range names {
			if n == "" || m[2*i] < 0 {
				continue
			}
			switch n {
			case "fn", "arrow", "method":
				name = string(src[m[2*i]:m[2*i+1]])
			}
		}
		if name == "" || tsControlKeywords[name] {
			continue
		}

		headerEnd := m[1] // end offset of the whole header match
		// Find the opening brace: the header match for fn/method ends right
		// after "(" so we must first skip to the matching ")" then the "{".
		// For the arrow case the match already ends just after "{".
		openBrace := headerEnd - 1
		if src[openBrace] != '{' {
			openBrace = findOpenBrace(src, headerEnd)
			if openBrace < 0 {
				continue
			}
		}
		closeBrace := findMatchingBrace(src, openBrace)
		if closeBrace < 0 {
			continue
		}

		startLine := 1 + strings.Count(string(src[:m[0]]), "\n")
		endLine := 1 + strings.Count(string(src[:closeBrace]), "\n")
		body := src[openBrace+1 : closeBrace]

		out = append(out, FuncInfo{
			File:      rel,
			Name:      name,
			StartLine: startLine,
			EndLine:   endLine,
			Shingles:  shingleTokens(tsTokens(body)),
		})
	}
	return out
}

// findOpenBrace scans forward from pos (just past the header's "(") to the
// matching ")" and then the following "{", skipping strings/templates so
// braces or parens inside them aren't miscounted.
func findOpenBrace(src []byte, pos int) int {
	depth := 1 // we start just after the header's opening "("
	i := pos
	for i < len(src) && depth > 0 {
		c := src[i]
		switch {
		case c == '"' || c == '\'' || c == '`':
			i = skipStringLiteral(src, i)
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			i = skipLineComment(src, i)
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			i = skipBlockComment(src, i)
			continue
		case c == '(':
			depth++
		case c == ')':
			depth--
		}
		i++
	}
	// Skip whitespace/return-type annotation up to the "{".
	for i < len(src) && src[i] != '{' {
		if src[i] == ';' || src[i] == '\n' && i > pos+200 {
			// Not a function we can extract a body for (e.g. an overload
			// signature or interface method); bail out.
		}
		i++
	}
	if i >= len(src) {
		return -1
	}
	return i
}

// findMatchingBrace returns the offset of the "}" matching the "{" at open,
// tracking string/template literals and comments so braces inside them
// don't unbalance the count.
func findMatchingBrace(src []byte, open int) int {
	depth := 0
	i := open
	for i < len(src) {
		c := src[i]
		switch {
		case c == '"' || c == '\'' || c == '`':
			i = skipStringLiteral(src, i)
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			i = skipLineComment(src, i)
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			i = skipBlockComment(src, i)
			continue
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// skipStringLiteral advances past a quoted/backtick string starting at i,
// honoring backslash escapes, and returns the offset just past its closing
// quote. Template literal `${...}` interpolation is treated as opaque text
// (good enough for brace-balancing purposes: nested braces inside `${}` are
// rare in the common case this heuristic targets).
func skipStringLiteral(src []byte, i int) int {
	quote := src[i]
	i++
	for i < len(src) {
		if src[i] == '\\' {
			i += 2
			continue
		}
		if src[i] == quote {
			return i + 1
		}
		i++
	}
	return i
}

func skipLineComment(src []byte, i int) int {
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i
}

func skipBlockComment(src []byte, i int) int {
	i += 2
	for i+1 < len(src) {
		if src[i] == '*' && src[i+1] == '/' {
			return i + 2
		}
		i++
	}
	return len(src)
}

// tsTokenRE is the language-neutral-ish lexer for JS/TS used to build
// shingles: identifiers normalize to "id", numeric/string/template literals
// normalize to "lit", and punctuation/operators are kept verbatim so
// structural differences (e.g. `if` vs `for`) still register.
var tsTokenRE = regexp.MustCompile(
	`[A-Za-z_$][\w$]*` + // identifier / keyword
		`|\d+\.?\d*` + // number literal
		`|"(?:[^"\\]|\\.)*"` + // double-quoted string
		"|`(?:[^`\\\\]|\\\\.)*`" + // template literal (backtick)
		`|'(?:[^'\\]|\\.)*'` + // single-quoted string
		`|=>|===|!==|==|!=|<=|>=|&&|\|\||\+\+|--|[{}()\[\];:,.<>+\-*/%=!&|^~?]`,
)

// tsTokens tokenizes a JS/TS source blob per tsTokenRE, normalizing
// identifiers and literals for the shingle helper in repoindex.go.
func tsTokens(src []byte) []string {
	raw := tsTokenRE.FindAll(src, -1)
	toks := make([]string, 0, len(raw))
	for _, t := range raw {
		switch t[0] {
		case '"', '\'', '`':
			toks = append(toks, "lit")
		default:
			if t[0] >= '0' && t[0] <= '9' {
				toks = append(toks, "lit")
			} else if isIdentByte(t[0]) {
				toks = append(toks, "id")
			} else {
				toks = append(toks, string(t))
			}
		}
	}
	return toks
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '$' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// shingleTokens builds the k-gram shingle set from an already-tokenized
// stream, mirroring repoindex.go's shingle() but for non-Go languages whose
// tokenizers live in this file / python.go rather than go/scanner.
func shingleTokens(toks []string) map[uint64]struct{} {
	set := map[uint64]struct{}{}
	if len(toks) == 0 {
		return set
	}
	if len(toks) < shingleK {
		set[hashTokens(toks)] = struct{}{}
		return set
	}
	for i := 0; i+shingleK <= len(toks); i++ {
		set[hashTokens(toks[i:i+shingleK])] = struct{}{}
	}
	return set
}
