package scope

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

func diff(paths ...string) *gitio.Diff {
	d := &gitio.Diff{}
	for _, p := range paths {
		d.Files = append(d.Files, gitio.FileChange{Path: p, Kind: gitio.KindModified, Added: 1})
	}
	return d
}

func TestScope_InScopePasses(t *testing.T) {
	g := New([]string{"src/auth/**"}, nil, false)
	fs, _ := g.Run(context.Background(), diff("src/auth/login.go", "src/auth/reset.go"), engine.NoIndex{})
	if len(fs) != 0 {
		t.Fatalf("in-scope changes should pass, got %+v", fs)
	}
}

func TestScope_OutOfScopeBlocked(t *testing.T) {
	g := New([]string{"src/auth/**"}, nil, false)
	fs, _ := g.Run(context.Background(), diff("src/auth/login.go", "pkg/util/unrelated.go"), engine.NoIndex{})
	if len(fs) != 1 || fs[0].File != "pkg/util/unrelated.go" {
		t.Fatalf("expected one out-of-scope finding for the util file, got %+v", fs)
	}
}

// always-allow globs (lockfiles, snapshots) are exempt from scope checks.
func TestScope_AllowAlwaysExempt(t *testing.T) {
	g := New([]string{"src/auth/**"}, []string{"**/*.lock"}, false)
	fs, _ := g.Run(context.Background(), diff("src/auth/login.go", "go.lock"), engine.NoIndex{})
	if len(fs) != 0 {
		t.Fatalf("lockfile should be exempt, got %+v", fs)
	}
}

// No manifest: warn-only by default (no findings) but blocks when required.
func TestScope_NoManifest(t *testing.T) {
	warn := New(nil, nil, false)
	if fs, _ := warn.Run(context.Background(), diff("anything.go"), engine.NoIndex{}); len(fs) != 0 {
		t.Errorf("no manifest + warn-only should not block, got %+v", fs)
	}
	strict := New(nil, nil, true)
	fs, _ := strict.Run(context.Background(), diff("anything.go"), engine.NoIndex{})
	if len(fs) != 1 || fs[0].RuleID != "no-task-manifest" {
		t.Errorf("no manifest + require should block, got %+v", fs)
	}
}
