package main

import (
	"os"
	"os/exec"
	"testing"
)

// newDoctorTestRepo makes a hermetic temp git repo (no global/system git
// config leakage) for exercising runDoctor without a subprocess.
func newDoctorTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

// TestRunDoctor_NoHooksInstalled asserts that a bare repo with no
// .dwarpal.yml and no hooks still passes the critical checks (git present,
// in a repo, config valid via defaults) even though the hooks check fails,
// because doctor's exit code reflects only the critical checks (PRD §5.1).
func TestRunDoctor_NoHooksInstalled(t *testing.T) {
	dir := newDoctorTestRepo(t)
	chdir(t, dir)

	err := runDoctor()
	if err != nil {
		t.Fatalf("runDoctor() = %v, want nil (critical checks should pass)", err)
	}
}

// TestRunDoctor_InvalidConfig asserts an invalid .dwarpal.yml is a critical
// failure: doctor must exit 2 so a broken config is never silently ignored.
func TestRunDoctor_InvalidConfig(t *testing.T) {
	dir := newDoctorTestRepo(t)
	if err := os.WriteFile(dir+"/.dwarpal.yml", []byte("mode: nonsense\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	err := runDoctor()
	if err == nil {
		t.Fatal("runDoctor() = nil, want error for invalid config")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("runDoctor() error type = %T, want *exitError", err)
	}
	if ee.code != 2 {
		t.Errorf("runDoctor() exit code = %d, want 2", ee.code)
	}
}

// TestRunDoctor_NoRepo asserts doctor fails critically (does not panic) when
// run outside any git work tree.
func TestRunDoctor_NoRepo(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	err := runDoctor()
	if err == nil {
		t.Fatal("runDoctor() = nil, want error outside a git repository")
	}
	ee, ok := err.(*exitError)
	if !ok {
		t.Fatalf("runDoctor() error type = %T, want *exitError", err)
	}
	if ee.code != 2 {
		t.Errorf("runDoctor() exit code = %d, want 2", ee.code)
	}
}
