package repoindex

import (
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/astengine"
)

// AST-backed function extraction for TS/JS/Python (tree-sitter-ast-engine
// change). The heuristic extractors in tsjs.go / python.go remain as the
// automatic fallback when a file fails to parse (design decision D4): a parse
// failure degrades to v1 behavior, never to a crash or silent skip.

// funcQueries captures each function-ish construct twice: the whole node
// (@fn, for the body/line range) and its name (@name).
var funcQueries = map[astengine.Language]string{
	astengine.LangTS: `[
  (function_declaration name: (identifier) @name) @fn
  (method_definition name: (property_identifier) @name) @fn
  (variable_declarator name: (identifier) @name value: (arrow_function)) @fn
]`,
	astengine.LangJS: `[
  (function_declaration name: (identifier) @name) @fn
  (method_definition name: (property_identifier) @name) @fn
  (variable_declarator name: (identifier) @name value: (arrow_function)) @fn
]`,
	astengine.LangPy: `(function_definition name: (identifier) @name) @fn`,
}

// functionsViaAST extracts FuncInfos through the tree-sitter engine. ok=false
// means the caller should use the heuristic extractor for this file.
func functionsViaAST(rel string, src []byte) ([]FuncInfo, bool) {
	lang := astengine.LanguageFor(rel)
	if lang == "" {
		return nil, false
	}
	tree, err := astengine.Parse(rel, src)
	if err != nil {
		return nil, false // degrade to heuristics (D4)
	}
	return functionsFromTree(tree, lang, rel, src)
}

// functionsFromTree runs function extraction over an already-parsed tree, so
// index building parses each file exactly once (shared with import counting).
func functionsFromTree(tree *astengine.Tree, lang astengine.Language, rel string, src []byte) ([]FuncInfo, bool) {
	query, ok := funcQueries[lang]
	if !ok {
		return nil, false
	}
	caps, err := tree.Query(query)
	if err != nil {
		return nil, false
	}

	// Captures arrive in node-position order, which puts each @fn (the whole
	// declaration) BEFORE its @name (the identifier inside it). Pair by
	// containment: a @name belongs to the most recent @fn whose line range
	// contains it.
	var funcs []FuncInfo
	for _, c := range caps {
		switch c.Name {
		case "fn":
			body := bodySlice(src, c.StartLine, c.EndLine)
			funcs = append(funcs, FuncInfo{
				File:      rel,
				StartLine: c.StartLine,
				EndLine:   c.EndLine,
				Shingles:  shingleTokens(tokensForLang(lang, body)),
			})
		case "name":
			for i := len(funcs) - 1; i >= 0; i-- {
				if funcs[i].Name == "" && c.StartLine >= funcs[i].StartLine && c.EndLine <= funcs[i].EndLine {
					funcs[i].Name = c.Text
					break
				}
			}
		}
	}

	// Partial tree (design D7): error regions swallowed some constructs (the
	// measured case: TS arrow functions with typed parameters). Supplement
	// with the heuristic extractor, merging by function name so AST-captured
	// entries keep their precise ranges.
	if tree.Partial {
		funcs = supplementHeuristic(funcs, lang, rel, src)
	}
	return funcs, true
}

// supplementHeuristic merges heuristic-extracted functions the AST tier
// missed (matched by name), keeping AST entries' precise ranges.
func supplementHeuristic(funcs []FuncInfo, lang astengine.Language, rel string, src []byte) []FuncInfo {
	seen := map[string]bool{}
	for _, f := range funcs {
		seen[f.Name] = true
	}
	var heur []FuncInfo
	if lang == astengine.LangPy {
		heur = FunctionsInPythonSource(rel, src)
	} else {
		heur = FunctionsInTSSource(rel, src)
	}
	for _, f := range heur {
		if !seen[f.Name] {
			funcs = append(funcs, f)
		}
	}
	return funcs
}

// tokensForLang reuses the language-neutral tokenizers the heuristic tier
// already ships, so AST-extracted and heuristic-extracted functions produce
// comparable shingles (a duplicate must match across extraction tiers).
func tokensForLang(lang astengine.Language, body []byte) []string {
	if lang == astengine.LangPy {
		return pyTokens(body)
	}
	return tsTokens(body)
}

// bodySlice returns the source lines [startLine, endLine] (1-indexed).
func bodySlice(src []byte, startLine, endLine int) []byte {
	lines := strings.Split(string(src), "\n")
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		return nil
	}
	return []byte(strings.Join(lines[startLine-1:endLine], "\n"))
}
