package duplicate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
	"github.com/YellowFoxH4XOR/dwarpal/internal/repoindex"
)

func TestDuplicate_FlagsNearDuplicate(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "existing.go"), []byte(`package p
func Alpha(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x * 2
	}
	return total
}
`), 0o644)
	// A near-identical function added in a new file.
	newFile := `package p
func Beta(ys []int) int {
	sum := 0
	for _, y := range ys {
		sum += y * 9
	}
	return sum
}
`
	os.WriteFile(filepath.Join(dir, "new.go"), []byte(newFile), 0o644)

	idx, err := repoindex.Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Mark the whole new file as added.
	var lines []gitio.Line
	for i := 1; i <= 8; i++ {
		lines = append(lines, gitio.Line{Number: i, Text: "x"})
	}
	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "new.go", Kind: gitio.KindAdded, AddedLines: lines}}}

	fs, err := New(dir, 0.8).Run(context.Background(), d, idx)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].RuleID != "no-duplicate-function" {
		t.Fatalf("expected one no-duplicate-function finding, got %+v", fs)
	}
	if fs[0].Line == 0 {
		t.Errorf("finding should point at the duplicate function's line")
	}
}

func TestDuplicate_NoIndexSkips(t *testing.T) {
	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "a.go", AddedLines: []gitio.Line{{Number: 1}}}}}
	// A not-ready index → gate is a safe no-op.
	fs, err := New("/nonexistent", 0.8).Run(context.Background(), d, &repoindex.Index{})
	if err != nil || len(fs) != 0 {
		t.Fatalf("no-index run should be a no-op, got findings=%d err=%v", len(fs), err)
	}
}
