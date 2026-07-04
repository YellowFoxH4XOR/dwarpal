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
	"time"

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
	// FuncByLang holds per-language function-convention counts, so drift and
	// analyze can compare against the norm FOR THAT LANGUAGE (snake_case is
	// normal in Python, camelCase in JS) instead of a Go-centric baseline.
	FuncByLang map[string]FuncStats
}

// FuncStats is one language's function-convention counts.
type FuncStats struct {
	Funcs          int
	SnakeCaseFuncs int
	TotalFuncLines int
}

// SnakeRatio returns lang's share of snake_case function names and the sample
// size, or (0,0) when the language has no counted functions.
func (c Conventions) SnakeRatio(lang string) (ratio float64, n int) {
	s := c.FuncByLang[lang]
	if s.Funcs == 0 {
		return 0, 0
	}
	return float64(s.SnakeCaseFuncs) / float64(s.Funcs), s.Funcs
}

// AvgFuncLinesFor returns lang's mean function length, or 0 when none counted.
func (c Conventions) AvgFuncLinesFor(lang string) float64 {
	s := c.FuncByLang[lang]
	if s.Funcs == 0 {
		return 0
	}
	return float64(s.TotalFuncLines) / float64(s.Funcs)
}

// addFuncStats accumulates funcs into lang's per-language counts. Cheap: it only
// inspects each function's already-extracted name and line span.
func (c *Conventions) addFuncStats(lang string, funcs []FuncInfo) {
	if lang == "" || len(funcs) == 0 {
		return
	}
	if c.FuncByLang == nil {
		c.FuncByLang = map[string]FuncStats{}
	}
	s := c.FuncByLang[lang]
	for _, fn := range funcs {
		s.Funcs++
		if strings.Contains(fn.Name, "_") {
			s.SnakeCaseFuncs++
		}
		s.TotalFuncLines += fn.EndLine - fn.StartLine + 1
	}
	c.FuncByLang[lang] = s
}

// LangLabel maps a path to the rule/convention language label used across gates
// ("go", "python", "typescript", "javascript"), or "" if unsupported.
func LangLabel(path string) string {
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
	return BuildFor(root, true)
}

// BuildFor builds the index with an explicit scope. needFuncs=false skips
// function extraction and shingling entirely — the drift gate only consumes
// the convention fingerprint, and on a 2,167-file TS repo the shingle work
// (plus its 267MB cache) was pure waste when duplicate detection is off.
func BuildFor(root string, needFuncs bool) (*Index, error) {
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
	// Whole-index deadline: per-file parse timeouts bound each file, but a
	// large repo of slow files could still sum to minutes. Past the deadline,
	// remaining files are indexed with the fast heuristic extractors instead
	// of tree-sitter — a degraded index beats a hung commit.
	deadline := time.Now().Add(indexDeadline)

	// Disk cache (#67): unchanged files (size+mtime match) skip parsing —
	// steady-state checks on large repos drop from seconds to milliseconds.
	cache := loadCache(root, needFuncs)
	fresh := make([]map[string]cacheEntry, workers)

	jobs := make(chan string)
	locals := make([]*Index, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		local := &Index{}
		locals[w] = local
		mine := map[string]cacheEntry{}
		fresh[w] = mine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rel := range jobs {
				abs := filepath.Join(root, rel)
				st, err := os.Stat(abs)
				if err != nil {
					continue
				}
				if e, ok := cache.Entries[rel]; ok && e.Size == st.Size() && e.MTime == st.ModTime().UnixNano() {
					funcs, conv := fromEntry(rel, e)
					local.Funcs = append(local.Funcs, funcs...)
					local.Conventions.merge(conv)
					mine[rel] = e
					continue
				}
				src, err := os.ReadFile(abs)
				if err != nil {
					continue
				}
				// Index into a scratch so this file's contribution is isolated
				// for caching, then fold it into the worker-local index.
				scratch := &Index{}
				scratch.addFileBudgeted(rel, src, needFuncs, time.Now().Before(deadline))
				local.Funcs = append(local.Funcs, scratch.Funcs...)
				local.Conventions.merge(scratch.Conventions)
				mine[rel] = toEntry(st.Size(), st.ModTime().UnixNano(), scratch.Funcs, scratch.Conventions)
			}
		}()
	}
	for _, rel := range paths {
		jobs <- rel
	}
	close(jobs)
	wg.Wait()

	idx := &Index{built: true}
	merged := cacheData{Entries: map[string]cacheEntry{}}
	for w, local := range locals {
		idx.Funcs = append(idx.Funcs, local.Funcs...)
		idx.Conventions.merge(local.Conventions)
		for rel, e := range fresh[w] {
			merged.Entries[rel] = e
		}
	}
	saveCache(root, merged, needFuncs)
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
	for lang, s := range o.FuncByLang {
		if c.FuncByLang == nil {
			c.FuncByLang = map[string]FuncStats{}
		}
		cur := c.FuncByLang[lang]
		cur.Funcs += s.Funcs
		cur.SnakeCaseFuncs += s.SnakeCaseFuncs
		cur.TotalFuncLines += s.TotalFuncLines
		c.FuncByLang[lang] = cur
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

// indexDeadline caps total tree-sitter time for one Build. Measured: synthetic
// corpora index in <1s, but real-world TS (observed live) can take 300ms+ per
// file even under the per-file timeout.
const indexDeadline = 5 * time.Second

// addFileBudgeted indexes one file. needFuncs=false skips function
// extraction (conventions only); astOK=false forces the heuristic tier
// (used once the whole-index deadline has passed).
func (idx *Index) addFileBudgeted(rel string, src []byte, needFuncs, astOK bool) {
	if !needFuncs && !strings.HasSuffix(rel, ".go") {
		// Conventions-only for non-Go: the import fingerprint needs a line
		// scan, not a parse. Function-convention counts use the HEURISTIC
		// (line-based) extractor too — no tree-sitter parse on this hot path,
		// so the p95 the hang fix protects stays intact.
		if ex := heuristicExtractorFor(rel); ex != nil {
			idx.Conventions.addFuncStats(LangLabel(rel), ex(rel, src))
		}
		idx.countImportsFromTree(nil, rel, src)
		return
	}
	if !astOK && !strings.HasSuffix(rel, ".go") {
		if ex := heuristicExtractorFor(rel); ex != nil {
			funcs := ex(rel, src)
			idx.Funcs = append(idx.Funcs, funcs...)
			idx.Conventions.addFuncStats(LangLabel(rel), funcs)
		}
		idx.countImportsFromTree(nil, rel, src) // line-scan path
		return
	}
	idx.addFile(rel, src)
}

// heuristicExtractorFor returns the non-AST extractor for a path.
func heuristicExtractorFor(path string) Extractor {
	switch {
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"),
		strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
		return FunctionsInTSSource
	case strings.HasSuffix(path, ".py"):
		return FunctionsInPythonSource
	default:
		return nil
	}
}

// addFile indexes one file with its language extractor. Go files additionally
// feed the convention fingerprint (drift is Go-only in v1).
func (idx *Index) addFile(rel string, src []byte) {
	if strings.HasSuffix(rel, ".go") {
		indexFile(idx, rel, src) // functions + conventions
		idx.countErrorIdioms(src)
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
	idx.Conventions.addFuncStats(LangLabel(rel), funcs)
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
		c.addFuncStats("go", []FuncInfo{info}) // per-language mirror
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
