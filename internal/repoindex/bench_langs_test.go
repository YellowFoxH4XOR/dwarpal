package repoindex

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Synthetic multi-language corpus benchmark (tree-sitter-ast-engine task 6.1).
// Parse cost is structural, so templated files measure throughput fairly.
// Gated behind an env var like the Go-stdlib benchmark: set BENCH_LANGS=1.
func TestBenchmark_MultiLanguageIndexBuild(t *testing.T) {
	if os.Getenv("BENCH_LANGS") == "" {
		t.Skip("BENCH_LANGS not set")
	}
	dir := t.TempDir()
	tsTmpl := `import { svc%d } from './svc';
export function handler%d(req: Request): Response {
  let total = 0;
  for (const item of req.items) {
    total += item.value * %d;
  }
  try {
    return svc%d.respond(total);
  } catch (e) {
    log.error(e);
    throw e;
  }
}
class Ctl%d {
  run(x: number): number {
    return x + %d;
  }
}
`
	pyTmpl := `import os
from typing import List

def process_%d(items: List[int]) -> int:
    total = 0
    for item in items:
        total += item * %d
    return total

class Handler%d:
    def run(self, x):
        try:
            return self.svc.call(x)
        except ValueError as e:
            raise
`
	const nTS, nPy = 1000, 500
	for i := 0; i < nTS; i++ {
		src := fmt.Sprintf(tsTmpl, i, i, i%7, i, i, i%13)
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%04d.ts", i)), []byte(src), 0o644)
	}
	for i := 0; i < nPy; i++ {
		src := fmt.Sprintf(pyTmpl, i, i%7, i)
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("p%04d.py", i)), []byte(src), 0o644)
	}

	start := time.Now()
	idx, err := Build(dir)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("BENCH: %d TS + %d Py files -> %d functions in %v (%.0f files/sec)",
		nTS, nPy, len(idx.Funcs), elapsed, float64(nTS+nPy)/elapsed.Seconds())
	if len(idx.Funcs) < nTS*2 { // handler + method per TS file at minimum
		t.Fatalf("extraction incomplete: %d funcs", len(idx.Funcs))
	}
	if !raceEnabled && elapsed > 2*time.Second {
		t.Errorf("index build %v exceeds the 2s pipeline budget", elapsed)
	}
}
