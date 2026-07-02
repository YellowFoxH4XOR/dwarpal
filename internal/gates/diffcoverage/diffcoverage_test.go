package diffcoverage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// diff builds a single-file Diff with the given added line numbers/text.
func diff(path string, lines ...int) *gitio.Diff {
	fc := gitio.FileChange{Path: path, Kind: gitio.KindModified}
	for _, ln := range lines {
		fc.AddedLines = append(fc.AddedLines, gitio.Line{Number: ln, Text: "x"})
		fc.Added++
	}
	return &gitio.Diff{Files: []gitio.FileChange{fc}}
}

// writeArtifact writes content to <dir>/<name> and returns name (relative to
// dir), so tests can exercise repoRoot-relative resolution.
func writeArtifact(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing artifact: %v", err)
	}
	return name
}

const lcovSample = `TN:
SF:src/pkg/foo.go
DA:1,1
DA:2,1
DA:3,0
DA:4,0
end_of_record
`

const goCoverSample = `mode: set
github.com/example/mod/src/pkg/foo.go:1.1,2.10 1 1
github.com/example/mod/src/pkg/foo.go:3.1,4.10 1 0
`

const coberturaSample = `<?xml version="1.0"?>
<coverage>
  <packages>
    <package name="pkg">
      <classes>
        <class name="foo" filename="src/pkg/foo.go">
          <lines>
            <line number="1" hits="1"/>
            <line number="2" hits="1"/>
            <line number="3" hits="0"/>
            <line number="4" hits="0"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>
`

func TestDiffCoverage_LCOV_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "lcov.info", lcovSample)
	g := New(name, 70, dir)

	// lines 1-4 added; 2/4 covered = 50% < 70% required.
	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2, 3, 4), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 1 || fs[0].RuleID != "below-threshold" || fs[0].Gate != gateID {
		t.Fatalf("expected one below-threshold finding, got %+v", fs)
	}
	if fs[0].Severity != "error" {
		t.Errorf("expected error severity, got %v", fs[0].Severity)
	}
}

func TestDiffCoverage_LCOV_AtThresholdPasses(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "lcov.info", lcovSample)
	g := New(name, 50, dir)

	// 2/4 = 50%, meets a 50% requirement.
	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2, 3, 4), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected pass at threshold, got %+v", fs)
	}
}

func TestDiffCoverage_LCOV_AboveThresholdPasses(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "lcov.info", lcovSample)
	g := New(name, 90, dir)

	// Only added lines 1 and 2 are both covered -> 100%.
	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected pass, got %+v", fs)
	}
}

func TestDiffCoverage_GoCover_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "cover.out", goCoverSample)
	g := New(name, 70, dir)

	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2, 3, 4), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 1 || fs[0].RuleID != "below-threshold" {
		t.Fatalf("expected below-threshold finding, got %+v", fs)
	}
}

func TestDiffCoverage_GoCover_AboveThresholdPasses(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "cover.out", goCoverSample)
	g := New(name, 40, dir)

	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2, 3, 4), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected pass, got %+v", fs)
	}
}

func TestDiffCoverage_Cobertura_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "coverage.xml", coberturaSample)
	g := New(name, 70, dir)

	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2, 3, 4), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 1 || fs[0].RuleID != "below-threshold" {
		t.Fatalf("expected below-threshold finding, got %+v", fs)
	}
}

func TestDiffCoverage_Cobertura_AboveThresholdPasses(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "coverage.xml", coberturaSample)
	g := New(name, 100, dir)

	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected pass, got %+v", fs)
	}
}

func TestDiffCoverage_MissingArtifact_WarnOnly(t *testing.T) {
	dir := t.TempDir()
	g := New("does-not-exist.info", 80, dir)

	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2), engine.NoIndex{})
	if err != nil {
		t.Fatalf("missing artifact must never error, got: %v", err)
	}
	for _, f := range fs {
		if f.Severity.Blocking() {
			t.Errorf("missing artifact must never block, got blocking finding %+v", f)
		}
	}
}

func TestDiffCoverage_MalformedArtifact_Errors(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "lcov.info", "this is not a coverage file at all\njust garbage\n")
	g := New(name, 70, dir)

	_, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2), engine.NoIndex{})
	if err == nil {
		t.Fatal("expected error for malformed/unrecognized artifact, got nil (must fail closed)")
	}
}

func TestDiffCoverage_MalformedCobertura_Errors(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "coverage.xml", "<?xml version=\"1.0\"?><coverage><packages>not closed")
	g := New(name, 70, dir)

	_, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2), engine.NoIndex{})
	if err == nil {
		t.Fatal("expected error for malformed cobertura XML, got nil (must fail closed)")
	}
}

func TestDiffCoverage_NoCoverableChangedLinesPasses(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "lcov.info", lcovSample)
	g := New(name, 90, dir)

	// File not present in coverage data at all -> nothing coverable -> pass.
	fs, err := g.Run(context.Background(), diff("src/pkg/untracked.go", 1, 2, 3), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected pass when no coverable changed lines, got %+v", fs)
	}
}

func TestDiffCoverage_ID(t *testing.T) {
	g := New("lcov.info", 70, "/repo")
	if g.ID() != "diff_coverage" {
		t.Fatalf("expected ID diff_coverage, got %q", g.ID())
	}
}

func TestDiffCoverage_AbsoluteArtifactPath(t *testing.T) {
	dir := t.TempDir()
	name := writeArtifact(t, dir, "lcov.info", lcovSample)
	abs := filepath.Join(dir, name)
	// repoRoot is intentionally a different, nonexistent directory: an
	// absolute artifactPath must not be joined onto it.
	g := New(abs, 50, "/nonexistent-repo-root")

	fs, err := g.Run(context.Background(), diff("src/pkg/foo.go", 1, 2, 3, 4), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected pass, got %+v", fs)
	}
}
