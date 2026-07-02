package drift

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
	"github.com/YellowFoxH4XOR/dwarpal/internal/repoindex"
)

// buildRepo writes a baseline of camelCase funcs plus a drift file, and returns
// the built index. The baseline is large so one snake_case outlier keeps the
// repo snake ratio below the 10% threshold (as at real-repo scale).
func buildRepo(t *testing.T, driftFile string) (string, *repoindex.Index) {
	t.Helper()
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("package p\n")
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "func camelCaseFn%d() int { return %d }\n", i, i)
	}
	os.WriteFile(filepath.Join(dir, "base.go"), []byte(b.String()), 0o644)
	os.WriteFile(filepath.Join(dir, "drift.go"), []byte(driftFile), 0o644)
	idx, err := repoindex.Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, idx
}

func TestDrift_NamingStyleOutlier(t *testing.T) {
	drift := "package p\nfunc my_snake_func() int {\n\treturn 1\n}\n"
	dir, idx := buildRepo(t, drift)
	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "drift.go", AddedLines: []gitio.Line{
		{Number: 2, Text: "func my_snake_func() int {"},
	}}}}

	fs, err := New(dir, "").Run(context.Background(), d, idx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range fs {
		if f.RuleID == "naming-style" && f.Severity == "info" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an info naming-style drift finding, got %+v", fs)
	}
}

func TestDrift_CleanCodeNoFindings(t *testing.T) {
	clean := "package p\nfunc anotherCamel() int {\n\treturn 1\n}\n"
	dir, idx := buildRepo(t, clean)
	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "drift.go", AddedLines: []gitio.Line{
		{Number: 2, Text: "func anotherCamel() int {"},
	}}}}
	fs, _ := New(dir, "").Run(context.Background(), d, idx)
	if len(fs) != 0 {
		t.Fatalf("clean camelCase code should not drift, got %+v", fs)
	}
}
