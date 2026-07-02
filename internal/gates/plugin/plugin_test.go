package plugin

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

func diffWith(paths ...string) *gitio.Diff {
	d := &gitio.Diff{}
	for _, p := range paths {
		d.Files = append(d.Files, gitio.FileChange{Path: p, Kind: gitio.KindModified, Added: 1})
	}
	return d
}

func TestPlugin_ExitZeroPasses(t *testing.T) {
	g := New("noop", "true", nil, "")
	fs, err := g.Run(context.Background(), diffWith("a.go"), engine.NoIndex{})
	if err != nil || len(fs) != 0 {
		t.Fatalf("exit 0 should pass with no findings: findings=%d err=%v", len(fs), err)
	}
}

func TestPlugin_NonzeroExitProducesFinding(t *testing.T) {
	g := New("failer", "echo problem found; exit 3", nil, "")
	fs, err := g.Run(context.Background(), diffWith("a.go"), engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 1 || fs[0].Gate != "plugin/failer" {
		t.Fatalf("want one finding from plugin/failer, got %+v", fs)
	}
	if fs[0].Suggestion == "" {
		t.Errorf("expected tool output captured in suggestion")
	}
}

// when globs gate whether the plugin runs at all.
func TestPlugin_WhenGlobSkips(t *testing.T) {
	g := New("py-only", "exit 1", []string{"**/*.py"}, "")
	// No python files changed → plugin does not run → no findings.
	fs, err := g.Run(context.Background(), diffWith("main.go"), engine.NoIndex{})
	if err != nil || len(fs) != 0 {
		t.Fatalf("plugin should be skipped: findings=%d err=%v", len(fs), err)
	}
	// A python file changed → plugin runs → nonzero exit → finding.
	fs, _ = g.Run(context.Background(), diffWith("app/x.py"), engine.NoIndex{})
	if len(fs) != 1 {
		t.Fatalf("plugin should run on .py change, got %d findings", len(fs))
	}
}

// A command that cannot start must fail closed (error, not silent pass).
func TestPlugin_UnstartableFailsClosed(t *testing.T) {
	g := New("bad", "this-binary-does-not-exist-xyz", nil, "")
	_, err := g.Run(context.Background(), diffWith("a.go"), engine.NoIndex{})
	// sh -c returns exit 127 for not-found, which is an ExitError → finding,
	// not a start error. Either way it must NOT silently pass.
	fs, _ := g.Run(context.Background(), diffWith("a.go"), engine.NoIndex{})
	if err == nil && len(fs) == 0 {
		t.Fatal("unstartable command must not silently pass")
	}
}
