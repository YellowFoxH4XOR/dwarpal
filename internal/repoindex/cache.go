package repoindex

import (
	"encoding/gob"
	"os"
	"path/filepath"
)

// Disk cache for the repo index (#67, promoted from deferred by live data:
// a 2,167-file TS repo took ~11s to index per check — every check — because
// nothing persisted). Entries are keyed by path and validated by size+mtime;
// only changed files re-parse, so steady-state checks skip tree-sitter almost
// entirely. The cache lives in .dwarpal/cache/ (gitignored) and is purely an
// accelerator: any read/decode problem falls back to a full build.

// Two cache scopes: the full index (with function shingles, for duplicate
// detection) and the conventions-only fingerprint (drift). gob, not JSON:
// shingle-heavy caches decoded 267MB of JSON in seconds; gob is compact and
// fast enough to be invisible.
func cacheFileName(needFuncs bool) string {
	if needFuncs {
		return "index-full.gob"
	}
	return "index-conv.gob"
}

// cacheEntry is one file's cached contribution to the index.
type cacheEntry struct {
	Size  int64      `json:"size"`
	MTime int64      `json:"mtime_ns"`
	Funcs []cachedFn `json:"funcs,omitempty"`
	// Conv is this file's fingerprint contribution (functions counted,
	// imports, idioms) so conventions merge identically from cache or parse.
	Conv Conventions `json:"conv"`
}

// cachedFn mirrors FuncInfo with shingles as a JSON-friendly slice.
type cachedFn struct {
	Name      string   `json:"name"`
	StartLine int      `json:"start"`
	EndLine   int      `json:"end"`
	Shingles  []uint64 `json:"sh"`
}

type cacheData struct {
	Entries map[string]cacheEntry `json:"entries"`
}

func cachePath(root string, needFuncs bool) string {
	return filepath.Join(root, ".dwarpal", "cache", cacheFileName(needFuncs))
}

// loadCache reads the cache; a missing or corrupt file is an empty cache.
func loadCache(root string, needFuncs bool) cacheData {
	data := cacheData{Entries: map[string]cacheEntry{}}
	f, err := os.Open(cachePath(root, needFuncs))
	if err != nil {
		return data
	}
	defer f.Close()
	if gob.NewDecoder(f).Decode(&data) != nil || data.Entries == nil {
		return cacheData{Entries: map[string]cacheEntry{}}
	}
	return data
}

// saveCache persists the cache best-effort — failing to write an accelerator
// must never fail the run.
func saveCache(root string, data cacheData, needFuncs bool) {
	path := cachePath(root, needFuncs)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(path), "idx-*")
	if err != nil {
		return
	}
	if gob.NewEncoder(f).Encode(data) != nil {
		f.Close()
		os.Remove(f.Name())
		return
	}
	f.Close()
	_ = os.Rename(f.Name(), path) // atomic: a torn cache must never exist
}

// toEntry converts a freshly-indexed file's contribution into a cache entry.
func toEntry(size int64, mtimeNS int64, funcs []FuncInfo, conv Conventions) cacheEntry {
	e := cacheEntry{Size: size, MTime: mtimeNS, Conv: conv}
	for _, f := range funcs {
		cf := cachedFn{Name: f.Name, StartLine: f.StartLine, EndLine: f.EndLine}
		for h := range f.Shingles {
			cf.Shingles = append(cf.Shingles, h)
		}
		e.Funcs = append(e.Funcs, cf)
	}
	return e
}

// fromEntry rehydrates a cache entry into index contributions.
func fromEntry(rel string, e cacheEntry) ([]FuncInfo, Conventions) {
	funcs := make([]FuncInfo, 0, len(e.Funcs))
	for _, cf := range e.Funcs {
		sh := make(map[uint64]struct{}, len(cf.Shingles))
		for _, h := range cf.Shingles {
			sh[h] = struct{}{}
		}
		funcs = append(funcs, FuncInfo{
			File: rel, Name: cf.Name,
			StartLine: cf.StartLine, EndLine: cf.EndLine, Shingles: sh,
		})
	}
	return funcs, e.Conv
}
