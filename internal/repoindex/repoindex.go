// Package repoindex builds the repo-level context the stateful gates need
// (duplicate-function, convention-drift). It resolves spike blocker B2 with a
// pragmatic v1 decision: use Go's standard-library go/parser for the AST tier
// rather than embed tree-sitter. This keeps the binary CGO-free and
// cross-compilable (the single-static-binary promise) at the cost of covering
// Go only in v1; TypeScript/Python grammars are a documented future change.
//
// The index implements engine.RepoIndex (structurally — Ready()), so gates
// receive it through the unchanged Gate signature and type-assert to *Index.
package repoindex

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// shingleK is the token n-gram size for function similarity.
const shingleK = 4

// FuncInfo describes one function in the repo.
type FuncInfo struct {
	File      string
	Name      string
	StartLine int
	EndLine   int
	Shingles  map[uint64]struct{}
}

// Conventions is a lightweight fingerprint of the repo's Go style, for drift.
type Conventions struct {
	Funcs          int
	ExportedFuncs  int
	SnakeCaseFuncs int // funcs whose names contain '_' (un-Go-like)
	TotalFuncLines int
}

// AvgFuncLines returns the mean function length, or 0 when empty.
func (c Conventions) AvgFuncLines() float64 {
	if c.Funcs == 0 {
		return 0
	}
	return float64(c.TotalFuncLines) / float64(c.Funcs)
}

// Index is the built repo index. Ready() satisfies engine.RepoIndex.
type Index struct {
	Funcs       []FuncInfo
	Conventions Conventions
	built       bool
}

// Ready reports whether the index was built (engine.RepoIndex).
func (i *Index) Ready() bool { return i != nil && i.built }

// skipDir names directories never worth indexing.
var skipDir = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, ".dwarpal": true,
}

// Build walks root, parses every .go file with go/parser, and indexes each
// function's shingles and convention stats. Parse errors on individual files
// are skipped (a syntactically broken file should not fail the whole index).
func Build(root string) (*Index, error) {
	idx := &Index{built: true}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		indexFile(idx, rel, src)
		return nil
	})
	return idx, err
}

// indexFile parses one Go source file and appends its functions to the index.
func indexFile(idx *Index, rel string, src []byte) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil || f == nil {
		return
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		start := fset.Position(fn.Body.Pos())
		end := fset.Position(fn.Body.End())
		body := src[start.Offset:end.Offset]
		info := FuncInfo{
			File:      rel,
			Name:      fn.Name.Name,
			StartLine: fset.Position(fn.Pos()).Line,
			EndLine:   end.Line,
			Shingles:  shingle(body),
		}
		idx.Funcs = append(idx.Funcs, info)

		c := &idx.Conventions
		c.Funcs++
		if fn.Name.IsExported() {
			c.ExportedFuncs++
		}
		if strings.Contains(fn.Name.Name, "_") {
			c.SnakeCaseFuncs++
		}
		c.TotalFuncLines += info.EndLine - info.StartLine + 1
	}
}

// FunctionsInSource parses a single Go source blob (a changed file from the
// working tree) into FuncInfos, for the duplicate/drift gates to compare
// changed functions against the index.
func FunctionsInSource(rel string, src []byte) []FuncInfo {
	tmp := &Index{}
	indexFile(tmp, rel, src)
	return tmp.Funcs
}

// shingle tokenizes Go source (via go/scanner) into k-gram token shingles,
// hashed to uint64. Comments and literal values are normalized out so
// near-duplicate logic with renamed strings still matches.
func shingle(src []byte) map[uint64]struct{} {
	var toks []string
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, src, nil, 0)
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		switch tok {
		case token.IDENT:
			toks = append(toks, "id") // normalize identifier names
		case token.INT, token.FLOAT, token.CHAR, token.STRING, token.IMAG:
			toks = append(toks, "lit") // normalize literals
		default:
			toks = append(toks, tok.String())
			_ = lit
		}
	}

	set := map[uint64]struct{}{}
	if len(toks) < shingleK {
		if len(toks) > 0 {
			set[hashTokens(toks)] = struct{}{}
		}
		return set
	}
	for i := 0; i+shingleK <= len(toks); i++ {
		set[hashTokens(toks[i:i+shingleK])] = struct{}{}
	}
	return set
}

func hashTokens(toks []string) uint64 {
	h := fnv.New64a()
	for _, t := range toks {
		h.Write([]byte(t))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

// Jaccard returns the similarity of two shingle sets in [0,1].
func Jaccard(a, b map[uint64]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	small, large := a, b
	if len(a) > len(b) {
		small, large = b, a
	}
	for k := range small {
		if _, ok := large[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
