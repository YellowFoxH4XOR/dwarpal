package repoindex

import (
	"os"
	"path/filepath"
	"testing"
)

// A warm (fully cached) rebuild must preserve everything duplicate detection
// needs — shingles survive the gob round-trip and still score as duplicates.
func TestCache_WarmRebuildPreservesShingles(t *testing.T) {
	dir := t.TempDir()
	src := `function sum(xs) {
  let t = 0;
  for (const x of xs) { t += x; }
  return t;
}
`
	os.WriteFile(filepath.Join(dir, "a.js"), []byte(src), 0o644)

	cold, err := BuildFor(dir, true)
	if err != nil || len(cold.Funcs) != 1 {
		t.Fatalf("cold build: funcs=%d err=%v", len(cold.Funcs), err)
	}
	if _, err := os.Stat(cachePath(dir, true)); err != nil {
		t.Fatal("cache file not written")
	}

	warm, err := BuildFor(dir, true)
	if err != nil || len(warm.Funcs) != 1 {
		t.Fatalf("warm build: funcs=%d err=%v", len(warm.Funcs), err)
	}
	if sim := Jaccard(cold.Funcs[0].Shingles, warm.Funcs[0].Shingles); sim != 1.0 {
		t.Fatalf("cached shingles differ from parsed: jaccard=%.2f", sim)
	}
	if warm.Funcs[0].File != "a.js" || warm.Funcs[0].Name != "sum" {
		t.Fatalf("cached identity wrong: %+v", warm.Funcs[0])
	}
}

// A modified file must be re-parsed, not served stale from cache.
func TestCache_ModifiedFileInvalidates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.py")
	os.WriteFile(path, []byte("def one():\n    return 1\n"), 0o644)
	if _, err := BuildFor(dir, true); err != nil {
		t.Fatal(err)
	}
	// Different size guarantees invalidation regardless of mtime granularity.
	os.WriteFile(path, []byte("def one():\n    return 1\n\ndef two():\n    return 2\n"), 0o644)
	idx, err := BuildFor(dir, true)
	if err != nil || len(idx.Funcs) != 2 {
		t.Fatalf("modified file not re-indexed: funcs=%d err=%v", len(idx.Funcs), err)
	}
}

// Conventions-only mode: no function extraction (no tree-sitter cost), but
// the import fingerprint still populates — what the drift gate needs.
func TestBuildFor_ConventionsOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.ts"), []byte("import { x } from 'y';\nfunction f() { return x; }\n"), 0o644)
	idx, err := BuildFor(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Funcs) != 0 {
		t.Fatalf("conventions-only must not extract functions, got %d", len(idx.Funcs))
	}
	if idx.Conventions.Imports["typescript"][FormESNamed] != 1 {
		t.Fatalf("import fingerprint missing: %+v", idx.Conventions.Imports)
	}
}
