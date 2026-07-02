package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// binDir holds the freshly built dwarpal binary made available on PATH to every
// testscript run. A real binary (not an in-process command) is required because
// the git hooks shell out to `dwarpal` from a /bin/sh subprocess.
var binDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dwarpal-bin")
	if err != nil {
		panic(err)
	}
	out := filepath.Join(dir, "dwarpal")
	cmd := exec.Command("go", "build", "-o", out, ".")
	if b, err := cmd.CombinedOutput(); err != nil {
		panic("building dwarpal for tests: " + err.Error() + "\n" + string(b))
	}
	binDir = dir

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// TestScripts runs the txtar acceptance scenarios in testdata/script. Each
// scenario maps to WHEN/THEN scenarios in the change's spec files.
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("testdata", "script"),
		Setup: func(e *testscript.Env) error {
			// Put the built binary first, then the host PATH so git resolves.
			e.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			// Deterministic git identity without touching host/global config.
			e.Setenv("GIT_AUTHOR_NAME", "Test")
			e.Setenv("GIT_AUTHOR_EMAIL", "test@dwarpal.dev")
			e.Setenv("GIT_COMMITTER_NAME", "Test")
			e.Setenv("GIT_COMMITTER_EMAIL", "test@dwarpal.dev")
			e.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			e.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
			// Default new repos to an agent/* branch so provenance detects an
			// agent (branch prefix) and the content gates apply — the realistic
			// path Dwarpal is built for. Human-skip is covered separately.
			e.Setenv("GIT_CONFIG_COUNT", "1")
			e.Setenv("GIT_CONFIG_KEY_0", "init.defaultBranch")
			e.Setenv("GIT_CONFIG_VALUE_0", "agent/main")
			return nil
		},
	})
}
