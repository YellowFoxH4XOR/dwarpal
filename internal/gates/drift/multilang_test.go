package drift

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
	"github.com/YellowFoxH4XOR/dwarpal/internal/repoindex"
)

// The heart of multi-language drift: in a Python repo (overwhelmingly
// snake_case), a newly added camelCase function is the outlier — the mirror
// image of the Go case. The old Go-centric rule (snake_case = drift) could
// never catch this; it would have flagged the CORRECT Python names instead.
func TestDrift_PythonFlagsCamelCase(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "def handle_request_%d():\n    return %d\n", i, i)
	}
	os.WriteFile(filepath.Join(dir, "base.py"), []byte(b.String()), 0o644)
	drift := "def getUserRecord():\n    return 1\n"
	os.WriteFile(filepath.Join(dir, "new.py"), []byte(drift), 0o644)

	idx, err := repoindex.Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "new.py", AddedLines: []gitio.Line{
		{Number: 1, Text: "def getUserRecord():"},
	}}}}

	fs, err := New(dir, "").Run(context.Background(), d, idx)
	if err != nil {
		t.Fatal(err)
	}
	if !hasNaming(fs, "camelCase") {
		t.Fatalf("a camelCase function in a snake_case Python repo must drift, got %+v", fs)
	}
}

// The correct-for-Python name must NOT drift — proving the baseline is learned
// per language, not hardcoded to Go's camelCase (which would wrongly flag this).
func TestDrift_PythonSnakeCaseClean(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, "def handle_request_%d():\n    return %d\n", i, i)
	}
	os.WriteFile(filepath.Join(dir, "base.py"), []byte(b.String()), 0o644)
	os.WriteFile(filepath.Join(dir, "new.py"), []byte("def handle_request_new():\n    return 1\n"), 0o644)

	idx, _ := repoindex.Build(dir)
	d := &gitio.Diff{Files: []gitio.FileChange{{Path: "new.py", AddedLines: []gitio.Line{
		{Number: 1, Text: "def handle_request_new():"},
	}}}}
	fs, _ := New(dir, "").Run(context.Background(), d, idx)
	for _, f := range fs {
		if f.RuleID == "naming-style" {
			t.Fatalf("a correct snake_case Python name must not drift: %+v", f)
		}
	}
}

// A single lowercase word (`run`) is valid under BOTH conventions and must
// never be flagged — the naming check keys on case markers, not absence of an
// underscore, so it doesn't false-positive on plain names.
func TestNamingDrift_LowercaseWordNeverFlagged(t *testing.T) {
	for _, ratio := range []float64{0.0, 1.0} {
		if rule, _ := namingDrift("python", ratio, "run"); rule != "" {
			t.Errorf("plain name 'run' must not drift at snakeRatio=%.0f", ratio)
		}
	}
}

func hasNaming(fs []finding.Finding, msgContains string) bool {
	for _, f := range fs {
		if f.RuleID == "naming-style" && strings.Contains(f.Message, msgContains) {
			return true
		}
	}
	return false
}
