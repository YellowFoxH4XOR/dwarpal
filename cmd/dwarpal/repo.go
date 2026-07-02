package main

import (
	"os/exec"
	"strings"
)

// repoRoot returns the top-level work-tree directory of the git repo containing
// cwd. A missing git binary or non-repo returns an error the caller maps to
// exit code 2.
func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gitAvailable reports whether the git binary is on PATH.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
