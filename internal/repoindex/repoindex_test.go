package repoindex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuild_IndexesFunctionsAndConventions(t *testing.T) {
	dir := t.TempDir()
	src := `package p

func Exported() int { return 1 }
func unexported() int { return 2 }
func snake_case_fn() int { return 3 }
`
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !idx.Ready() {
		t.Fatal("index should be ready")
	}
	if idx.Conventions.Funcs != 3 {
		t.Errorf("funcs = %d, want 3", idx.Conventions.Funcs)
	}
	if idx.Conventions.ExportedFuncs != 1 {
		t.Errorf("exported = %d, want 1", idx.Conventions.ExportedFuncs)
	}
	if idx.Conventions.SnakeCaseFuncs != 1 {
		t.Errorf("snake_case = %d, want 1", idx.Conventions.SnakeCaseFuncs)
	}
}

// Two structurally identical functions (different names/literals) must score as
// near-duplicates; an unrelated function must not.
func TestJaccard_NearDuplicateDetection(t *testing.T) {
	a := FunctionsInSource("a.go", []byte(`package p
func Alpha(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x * 2
	}
	return total
}`))
	b := FunctionsInSource("b.go", []byte(`package p
func Beta(ys []int) int {
	sum := 0
	for _, y := range ys {
		sum += y * 5
	}
	return sum
}`))
	c := FunctionsInSource("c.go", []byte(`package p
func Gamma(s string) bool {
	return len(s) > 0 && s[0] == 'x'
}`))

	dup := Jaccard(a[0].Shingles, b[0].Shingles)
	if dup < 0.8 {
		t.Errorf("renamed-but-identical funcs should be near-duplicates, jaccard=%.2f", dup)
	}
	unrelated := Jaccard(a[0].Shingles, c[0].Shingles)
	if unrelated > 0.3 {
		t.Errorf("unrelated funcs should score low, jaccard=%.2f", unrelated)
	}
}

func TestBuild_SkipsBrokenFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\nfunc F() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package p\nfunc {{{"), 0o644)
	idx, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	// The broken file is skipped; the good one is still indexed.
	if idx.Conventions.Funcs != 1 {
		t.Errorf("funcs = %d, want 1 (broken file skipped)", idx.Conventions.Funcs)
	}
}
