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

// Import-style dimension: require() in a named-import TS repo is flagged;
// matching style is not; a weak majority flags nothing. Also proves the
// dimension works with ZERO Go functions in the repo (guard scoping).
func TestDrift_ImportStyle(t *testing.T) {
	dir := t.TempDir()
	idx := &repoindex.Index{}
	_ = idx // conventions built by hand below for exact shares
	conv := repoindex.Conventions{Imports: map[string]map[string]int{
		"typescript": {"es-named": 9, "require": 1}, // 90% named
	}}
	built, err := repoindex.Build(dir) // empty repo -> Ready() index
	if err != nil {
		t.Fatal(err)
	}
	built.Conventions = conv

	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "new.ts", AddedLines: []gitio.Line{
		{Number: 1, Text: `const x = require('y');`},
		{Number: 2, Text: `import { z } from 'z';`},
	}}}}
	fs, err := New(dir, "").Run(context.Background(), d, built)
	if err != nil {
		t.Fatal(err)
	}
	var importFindings int
	for _, f := range fs {
		if f.RuleID == "import-style" {
			importFindings++
			if f.Line != 1 {
				t.Fatalf("finding should be on the require line, got %+v", f)
			}
		}
	}
	if importFindings != 1 {
		t.Fatalf("want exactly 1 import-style finding, got %d (%+v)", importFindings, fs)
	}

	// Weak majority (60/40): no norm, no findings.
	built.Conventions = repoindex.Conventions{Imports: map[string]map[string]int{
		"typescript": {"es-named": 6, "require": 4},
	}}
	fs, _ = New(dir, "").Run(context.Background(), d, built)
	for _, f := range fs {
		if f.RuleID == "import-style" {
			t.Fatalf("weak majority must not flag, got %+v", f)
		}
	}
}
