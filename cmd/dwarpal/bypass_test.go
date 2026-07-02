package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir switches the process cwd to dir for the duration of the test, since
// runBypass (via repoRoot) resolves the repo from the process cwd.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRunBypass_WritesLogLine(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "hello\n")
	gitCmd(t, dir, "git", "add", ".")
	gitCmd(t, dir, "git", "commit", "-m", "init")
	chdir(t, dir)

	if err := runBypass("urgent hotfix, reviewed offline"); err != nil {
		t.Fatalf("runBypass: %v", err)
	}

	logPath := filepath.Join(dir, bypassLogDir, bypassLogFile)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading bypass log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("bypass log lines = %d, want 1 (%q)", len(lines), string(data))
	}
	if !strings.Contains(lines[0], "urgent hotfix") {
		t.Errorf("bypass log line missing reason: %q", lines[0])
	}
}

func TestRunBypass_RequiresReason(t *testing.T) {
	dir := newRepo(t)
	chdir(t, dir)

	err := runBypass("")
	ee, ok := err.(*exitError)
	if !ok || ee.code != 2 {
		t.Fatalf("runBypass(\"\") err = %v, want exitError{code:2}", err)
	}
}

func TestRunBypass_RejectedInCIStrict(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, ".dwarpal.yml", "mode: ci_strict\n")
	gitCmd(t, dir, "git", "add", ".")
	gitCmd(t, dir, "git", "commit", "-m", "init")
	chdir(t, dir)

	err := runBypass("please let me through")
	ee, ok := err.(*exitError)
	if !ok || ee.code != 2 {
		t.Fatalf("runBypass under ci_strict err = %v, want exitError{code:2}", err)
	}

	if _, statErr := os.Stat(filepath.Join(dir, bypassLogDir, bypassLogFile)); !os.IsNotExist(statErr) {
		t.Errorf("bypass.log should not be written when ci_strict rejects the bypass")
	}
}
