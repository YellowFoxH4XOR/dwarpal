// Package repoindex builds the repo-level context the stateful gates need
// (duplicate-function, convention-drift). Go files use the stdlib go/parser
// (a true AST, fastest for Go); TypeScript/JavaScript/Python use the pure-Go
// tree-sitter runtime via internal/astengine, with the original heuristic
// extractors as automatic fallback on parse failure. Everything stays
// CGO-free, preserving the single-static-binary promise.
//
// The index implements engine.RepoIndex (structurally — Ready()), so gates
// receive it through the unchanged Gate signature and type-assert to *Index.
package repoindex

import (
	"go/ast"
	"runtime"
	"sync"

	"github.com/YellowFoxH4XOR/dwarpal/internal/astengine"
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

// Conventions is a lightweight fingerprint of the repo's style, for drift.
type Conventions struct {
	Funcs          int
	ExportedFuncs  int
	SnakeCaseFuncs int // funcs whose names contain '_' (un-Go-like)
	TotalFuncLines int
	// Imports counts import forms per language (lang -> form -> count),
	// consumed by the drift gate's import-style dimension.
	Imports map[string]map[string]int
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

// Build walks root and indexes every supported file's functions, shingles,
// and convention stats. Files are independent, so parsing fans out across
// CPU cores into worker-local indexes merged at the end — the multi-language
// benchmark showed serial parsing alone blowing the 2s pipeline budget.
// Parse errors on individual files are skipped (a syntactically broken file
// should not fail the whole index).
func Build(root string) (*Index, error) {
	var paths []string
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
		rel, _ := filepath.Rel(root, path)
		if FunctionsFor(rel) == nil {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return &Index{built: true}, err
	}

	workers := runtime.NumCPU()
	if workers > len(paths) {
		workers = len(paths)
	}
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string)
	locals := make([]*Index, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		local := &Index{}
		locals[w] = local
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				src, err := os.ReadFile(filepath.Join(root, rel))
				if err != nil {
					continue
				}
				local.addFile(rel, src)
			}
		}()
	}
	for _, rel := range paths {
		jobs <- rel
	}
	close(jobs)
	wg.Wait()

	idx := &Index{built: true}
	for _, local := range locals {
		idx.Funcs = append(idx.Funcs, local.Funcs...)
		idx.Conventions.merge(local.Conventions)
	}
	return idx, nil
}

// merge folds another fingerprint into this one (worker-local index merge).
func (c *Conventions) merge(o Conventions) {
	c.Funcs += o.Funcs
	c.ExportedFuncs += o.ExportedFuncs
	c.SnakeCaseFuncs += o.SnakeCaseFuncs
	c.TotalFuncLines += o.TotalFuncLines
	for lang, forms := range o.Imports {
		for form, n := range forms {
			if c.Imports == nil {
				c.Imports = map[string]map[string]int{}
			}
			if c.Imports[lang] == nil {
				c.Imports[lang] = map[string]int{}
			}
			c.Imports[lang][form] += n
		}
	}
}

// Extractor turns one source file into its function inventory entries.
type Extractor func(rel string, src []byte) []FuncInfo

// FunctionsFor returns the extractor for a path's language, or nil when the
// language has no support. Go uses the stdlib go/parser; TS/JS and Python go
// AST-first (tree-sitter) with heuristic fallback.
func FunctionsFor(path string) Extractor {
	switch {
	case strings.HasSuffix(path, ".go"):
		return FunctionsInSource
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"),
		strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
		return astFirst(FunctionsInTSSource)
	case strings.HasSuffix(path, ".py"):
		return astFirst(FunctionsInPythonSource)
	default:
		return nil
	}
}

// astFirst prefers tree-sitter extraction and degrades to the heuristic
// extractor when parsing fails (design D4) — so a grammar bug or pathological
// file costs precision for that one file, never correctness of the run.
func astFirst(fallback Extractor) Extractor {
	return func(rel string, src []byte) []FuncInfo {
		if funcs, ok := functionsViaAST(rel, src); ok {
			return funcs
		}
		return fallback(rel, src)
	}
}

// addFile indexes one file with its language extractor. Go files additionally
// feed the convention fingerprint (drift is Go-only in v1).
func (idx *Index) addFile(rel string, src []byte) {
	if strings.HasSuffix(rel, ".go") {
		indexFile(idx, rel, src) // functions + conventions
		return
	}
	// Parse once; share the tree between function extraction and import
	// counting (the multi-language benchmark showed double-parsing plus
	// per-file query recompilation dominated the index cost).
	lang := astengine.LanguageFor(rel)
	tree, err := astengine.Parse(rel, src)
	if err != nil {
		tree = nil
	}
	var funcs []FuncInfo
	extracted := false
	if tree != nil {
		funcs, extracted = functionsFromTree(tree, lang, rel, src)
		if extracted && tree.Partial {
			funcs = supplementHeuristic(funcs, lang, rel, src)
		}
	}
	if !extracted {
		if ex := FunctionsFor(rel); ex != nil {
			funcs = ex(rel, src)
		}
	}
	idx.Funcs = append(idx.Funcs, funcs...)
	idx.countImportsFromTree(tree, rel, src)
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
