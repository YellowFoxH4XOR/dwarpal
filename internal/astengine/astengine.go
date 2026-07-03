// Package astengine is Dwarpal's single seam over the tree-sitter runtime.
//
// Design decision D2 (tree-sitter-ast-engine change): gates and the repo index
// never import the third-party parser directly — they call Supports/Parse/
// Query here. If the dependency ever needs replacing, one package changes.
//
// The runtime is github.com/odvcencio/gotreesitter: a pure-Go tree-sitter
// reimplementation (no cgo), which is what keeps the §5.5 single-static-binary
// promise intact while giving real syntax trees for TS/JS/Python. Go files are
// deliberately NOT parsed here — the stdlib go/parser (already used by
// repoindex and archrules) is a true AST and faster for Go.
package astengine

import (
	"fmt"
	"strings"
	"sync"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Language identifies a supported AST language.
type Language string

const (
	LangTS Language = "typescript"
	LangJS Language = "javascript"
	LangPy Language = "python"
)

// LanguageFor returns the AST language for a path, or "" when the file is
// outside the registry (decision D3: the registry is the single authority;
// everything else falls through to heuristic behavior).
func LanguageFor(path string) Language {
	switch {
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return LangTS
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
		return LangJS
	case strings.HasSuffix(path, ".py"):
		return LangPy
	default:
		return ""
	}
}

// Supports reports whether path's language has tree-sitter support here.
func Supports(path string) bool { return LanguageFor(path) != "" }

// Tree is a parsed file, ready for queries. Partial marks a tree that parsed
// with localized errors (design D7): captures from it come only from
// structurally valid regions, but constructs inside error regions are missing
// — callers should supplement with the heuristic tier.
type Tree struct {
	bound   *ts.BoundTree
	lang    *ts.Language
	grammar string // detected grammar identity (e.g. "typescript" vs "tsx")
	src     []byte
	Partial bool
}

// Parse parses src as path's language. An error means the caller should fall
// back to the heuristic tier for this file (decision D4) — never fail the run.
func Parse(path string, src []byte) (*Tree, error) {
	if LanguageFor(path) == "" {
		return nil, fmt.Errorf("astengine: unsupported language for %s", path)
	}
	// Use the SAME grammar the parser will pick (DetectLanguage): .tsx maps to
	// the TSX grammar, whose node IDs differ from plain TypeScript — compiling
	// queries against the wrong grammar silently matches nothing (found by the
	// realistic-code check: a .tsx component parsed fine but extracted zero
	// functions).
	entry := grammars.DetectLanguage(path)
	if entry == nil {
		return nil, fmt.Errorf("astengine: no grammar for %s", path)
	}
	lang := entry.Language()
	bound, err := grammars.ParseFile(path, src)
	if err != nil {
		return nil, fmt.Errorf("astengine: parsing %s: %w", path, err)
	}
	root := bound.RootNode()
	if root == nil {
		return nil, fmt.Errorf("astengine: %s produced no tree", path)
	}
	// Tolerant mode (design D7): trees with errors — even an ERROR root — still
	// hold recovered, queryable subtrees with accurate positions (verified:
	// a method capture inside an ERROR root reports the right line). Query
	// captures only match structurally valid regions, so we keep the tree and
	// set Partial, signalling callers to supplement misses (e.g. the known
	// typed-arrow grammar gap in the TS grammar) with the heuristic tier.
	// A root with NO recovered children is useless — degrade entirely.
	if root.IsError() && root.ChildCount() == 0 {
		return nil, fmt.Errorf("astengine: %s failed to parse", path)
	}
	return &Tree{bound: bound, lang: lang, grammar: entry.Name, src: src, Partial: root.HasError()}, nil
}

// Capture is one query capture: its capture name, source text, and 1-indexed
// line range — everything a finding or FuncInfo needs.
type Capture struct {
	Name      string
	Text      string
	StartLine int
	EndLine   int
}

// queryCache holds compiled queries keyed by (language, source). Compiling a
// query is far more expensive than executing it; without this cache an index
// build recompiled every query for every file (measured: the dominant cost of
// the multi-language benchmark).
var (
	queryMu    sync.RWMutex
	queryCache = map[string]*ts.Query{}
)

func compiledQuery(lang *ts.Language, grammar string, query string) (*ts.Query, error) {
	// Key by the detected GRAMMAR, not the coarse language label: .ts and .tsx
	// share a label but use different grammars with different node IDs — a
	// cache collision here silently matches nothing (found by the TSX
	// regression test).
	key := grammar + "\x00" + query
	queryMu.RLock()
	q, ok := queryCache[key]
	queryMu.RUnlock()
	if ok {
		return q, nil
	}
	q, err := ts.NewQuery(query, lang)
	if err != nil {
		return nil, err
	}
	queryMu.Lock()
	queryCache[key] = q
	queryMu.Unlock()
	return q, nil
}

// Query compiles (cached) and runs a tree-sitter .scm query, returning all
// captures in document order.
func (t *Tree) Query(query string) ([]Capture, error) {
	q, err := compiledQuery(t.lang, t.grammar, query)
	if err != nil {
		return nil, fmt.Errorf("astengine: compiling query: %w", err)
	}
	cur := q.Exec(t.bound.RootNode(), t.lang, t.src)
	var out []Capture
	for {
		m, ok := cur.NextMatch()
		if !ok {
			break
		}
		for _, c := range m.Captures {
			text := c.TextOverride
			if text == "" {
				text = string(t.src[c.Node.StartByte():c.Node.EndByte()])
			}
			out = append(out, Capture{
				Name:      c.Name,
				Text:      text,
				StartLine: int(c.Node.StartPoint().Row) + 1,
				EndLine:   int(c.Node.EndPoint().Row) + 1,
			})
		}
	}
	return out, nil
}
