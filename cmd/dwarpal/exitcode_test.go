package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// dwarpalBin is the binary built in TestMain.
func dwarpalBin() string { return filepath.Join(binDir, "dwarpal"+binExt) }

// newRepo makes a temp git repo with deterministic identity.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitCmd(t, dir, "git", "init")
	return dir
}

func gitCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = repoEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func repoEnv() []string {
	return append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t.co",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t.co",
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull,
		// New repos default to an agent/* branch so provenance detects an agent
		// and the content gates apply (default apply_gates_to: agent-only).
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=init.defaultBranch", "GIT_CONFIG_VALUE_0=agent/main")
}

// checkExit runs `dwarpal check` in dir and returns its exit code.
func checkExit(t *testing.T, dir string, args ...string) int {
	t.Helper()
	cmd := exec.Command(dwarpalBin(), append([]string{"check"}, args...)...)
	cmd.Dir = dir
	cmd.Env = repoEnv()
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if ok := asExitError(err, &ee); ok {
		return ee.ExitCode()
	}
	t.Fatalf("unexpected error running check: %v", err)
	return -1
}

func asExitError(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// Exit codes are a contract: 0 pass, 1 blocked, 2 config/internal error.
func TestExitCodes(t *testing.T) {
	// 0 — within budget
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "one\ntwo\n")
	gitCmd(t, dir, "git", "add", ".")
	if code := checkExit(t, dir); code != 0 {
		t.Errorf("passing check exit = %d, want 0", code)
	}

	// 1 — over budget
	dir = newRepo(t)
	writeFile(t, dir, ".dwarpal.yml", "gates:\n  diff_budget:\n    max_lines: 2\n")
	writeFile(t, dir, "big.txt", "a\nb\nc\nd\n")
	gitCmd(t, dir, "git", "add", ".")
	if code := checkExit(t, dir); code != 1 {
		t.Errorf("blocked check exit = %d, want 1", code)
	}

	// 2 — invalid config
	dir = newRepo(t)
	writeFile(t, dir, ".dwarpal.yml", "mode: nonsense\n")
	if code := checkExit(t, dir); code != 2 {
		t.Errorf("invalid-config exit = %d, want 2", code)
	}
}

// M0 exit criterion (PRD §10): a 600-line staged diff is blocked in under 1s.
func TestM0_OversizedDiffBlockedUnderOneSecond(t *testing.T) {
	dir := newRepo(t)
	var sb strings.Builder
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	writeFile(t, dir, "big.go", sb.String())
	gitCmd(t, dir, "git", "add", ".")

	start := time.Now()
	code := checkExit(t, dir)
	elapsed := time.Since(start)

	if code != 1 {
		t.Errorf("600-line diff exit = %d, want 1 (blocked)", code)
	}
	if elapsed > time.Second {
		t.Errorf("check took %v, want < 1s (M0 budget)", elapsed)
	}
	t.Logf("M0: 600-line diff blocked in %v", elapsed)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
