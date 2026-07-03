package repoindex

import (
	"os"
	"testing"
	"time"
)

// Measures the eager index build on a very large real corpus (Go stdlib src).
func TestBenchmark_LargeRepoIndexBuild(t *testing.T) {
	corpus := os.Getenv("BENCH_CORPUS")
	if corpus == "" {
		t.Skip("BENCH_CORPUS not set")
	}
	start := time.Now()
	idx, err := Build(corpus)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("BENCH: %d functions indexed in %v (%.0f funcs/sec)",
		idx.Conventions.Funcs, elapsed, float64(idx.Conventions.Funcs)/elapsed.Seconds())
}
