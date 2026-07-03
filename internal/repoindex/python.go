// Heuristic Python function extraction: no full AST (same v1 tradeoff as
// tsjs.go — see repoindex.go's package doc), so functions are found by
// matching `def name(` at some indentation and taking the body as the run
// of subsequent lines indented deeper than the `def` line, per Python's own
// indentation-defines-blocks rule (far simpler and more reliable for Python
// than brace-balancing is for TS/JS).
//
// Documented choice on nesting: both the outer function and any nested
// `def` inside it are extracted as separate FuncInfo entries. The outer
// function's shingles/body include the nested def's source (so drift/dup
// comparisons of the outer function still see its full behavior), and the
// inner def is also indexed on its own so a helper duplicated across two
// outer functions is still caught.
package repoindex

import (
	"regexp"
	"strings"
)

// pyDefRE matches a `def name(` line and captures its leading indentation
// (spaces/tabs) and the function name.
var pyDefRE = regexp.MustCompile(`^([ \t]*)def[ \t]+([A-Za-z_]\w*)[ \t]*\(`)

// FunctionsInPythonSource heuristically extracts functions from a Python
// source blob. StartLine is the `def` line; EndLine is the last line whose
// indentation is deeper than the def line (i.e. the last line still inside
// the body), so blank/comment-only trailing lines aren't miscounted against
// a dedent that hasn't happened yet.
func FunctionsInPythonSource(rel string, src []byte) []FuncInfo {
	lines := strings.Split(string(src), "\n")

	var out []FuncInfo
	for i, line := range lines {
		m := pyDefRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent := indentWidth(m[1])
		name := m[2]

		endLine := i + 1 // 1-based line of the def itself, updated below
		bodyStart := i
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "" {
				continue // blank lines don't end the body
			}
			if indentWidth(leadingWhitespace(lines[j])) <= indent {
				break
			}
			endLine = j + 1
		}
		if endLine == i+1 && bodyStart == i {
			// No indented body followed (e.g. a stub/last line in file);
			// still index the single def line so callers see it exists.
		}

		body := strings.Join(lines[i:endLine], "\n")
		out = append(out, FuncInfo{
			File:      rel,
			Name:      name,
			StartLine: i + 1,
			EndLine:   endLine,
			Shingles:  shingleTokens(pyTokens([]byte(body))),
		})
	}
	return out
}

// leadingWhitespace returns the leading run of spaces/tabs on a line.
func leadingWhitespace(line string) string {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i]
}

// indentWidth counts indentation width, treating a tab as 8 columns (matches
// Python's own tokenizer convention closely enough for relative comparisons).
func indentWidth(ws string) int {
	w := 0
	for _, c := range ws {
		if c == '\t' {
			w += 8
		} else {
			w++
		}
	}
	return w
}

// pyTokenRE is the lexer used to build shingles for Python: identifiers/
// keywords normalize to "id", numeric/string literals normalize to "lit",
// and operators/punctuation are kept verbatim.
var pyTokenRE = regexp.MustCompile(
	`[A-Za-z_]\w*` + // identifier / keyword
		`|\d+\.?\d*` + // number literal
		`|"""(?:[^\\]|\\.)*?"""` + // triple double-quoted string
		`|'''(?:[^\\]|\\.)*?'''` + // triple single-quoted string
		`|"(?:[^"\\\n]|\\.)*"` + // double-quoted string
		`|'(?:[^'\\\n]|\\.)*'` + // single-quoted string
		`|==|!=|<=|>=|\*\*|//|->|[{}()\[\]:,.+\-*/%=<>!&|^~]`,
)

// pyTokens tokenizes a Python source blob per pyTokenRE, normalizing
// identifiers and literals for the shingle helper in tsjs.go.
func pyTokens(src []byte) []string {
	raw := pyTokenRE.FindAll(src, -1)
	toks := make([]string, 0, len(raw))
	for _, t := range raw {
		switch {
		case t[0] == '"' || t[0] == '\'':
			toks = append(toks, "lit")
		case t[0] >= '0' && t[0] <= '9':
			toks = append(toks, "lit")
		case t[0] == '_' || (t[0] >= 'a' && t[0] <= 'z') || (t[0] >= 'A' && t[0] <= 'Z'):
			toks = append(toks, "id")
		default:
			toks = append(toks, string(t))
		}
	}
	return toks
}
