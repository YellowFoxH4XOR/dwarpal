package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
)

// wantHooksPath is the core.hooksPath value Dwarpal sets on install (see
// internal/hooks.dirRel), duplicated here since hooks.dirRel is unexported.
const wantHooksPath = ".dwarpal/hooks"

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose Dwarpal's setup in this repo",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDoctor()
		},
	}
}

// runDoctor runs the PRD §5.1 diagnostics and prints a ✓/✗ line per check.
// It exits 0 if all critical checks pass (git present, in a repo, config
// valid) regardless of the non-critical checks, else exits 2.
func runDoctor() error {
	critical := true

	if ok, detail := checkGit(); ok {
		fmt.Printf("✓ git: %s\n", detail)
	} else {
		fmt.Printf("✗ git: %s\n", detail)
		critical = false
	}

	root, err := repoRoot()
	if err != nil {
		fmt.Printf("✗ git repository: %s\n", err.Error())
		critical = false
	} else {
		fmt.Printf("✓ git repository: %s\n", root)
	}

	if root != "" {
		if _, err := config.Load(root); err != nil {
			fmt.Printf("✗ %s: %s\n", config.Filename, err.Error())
			critical = false
		} else {
			fmt.Printf("✓ %s: valid (or absent — defaults apply)\n", config.Filename)
		}

		if ok, detail := checkHooks(root); ok {
			fmt.Printf("✓ hooks: %s\n", detail)
		} else {
			fmt.Printf("✗ hooks: %s\n", detail)
		}
	} else {
		fmt.Println("✗ .dwarpal.yml: skipped (no git repository)")
		fmt.Println("✗ hooks: skipped (no git repository)")
	}

	fmt.Println("✓ AST language support: Go (stdlib go/parser)")
	fmt.Println("✗ AST language support: TypeScript/Python: not in this build")

	if !critical {
		return &exitError{code: 2, msg: "doctor found critical issues"}
	}
	return nil
}

// checkGit reports whether the system git binary is on PATH, with its version.
func checkGit() (bool, string) {
	path, err := exec.LookPath("git")
	if err != nil {
		return false, "not found on PATH"
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return false, fmt.Sprintf("found at %s but failed to run: %s", path, err)
	}
	return true, strings.TrimSpace(string(out))
}

// checkHooks reports whether Dwarpal's hooks are installed: core.hooksPath
// points at .dwarpal/hooks and the pre-commit/pre-push scripts exist and are
// executable.
func checkHooks(root string) (bool, string) {
	cmd := exec.Command("git", "config", "--local", "--get", "core.hooksPath")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false, "core.hooksPath not set (run 'dwarpal init' or 'dwarpal hook install')"
	}
	hooksPath := strings.TrimSpace(string(out))
	if hooksPath != wantHooksPath {
		return false, fmt.Sprintf("core.hooksPath is %q, want %q", hooksPath, wantHooksPath)
	}

	hooksDir := filepath.Join(root, wantHooksPath)
	for _, name := range []string{"pre-commit", "pre-push"} {
		p := filepath.Join(hooksDir, name)
		fi, err := os.Stat(p)
		if err != nil {
			return false, fmt.Sprintf("%s: missing", name)
		}
		if fi.Mode()&0o111 == 0 {
			return false, fmt.Sprintf("%s: not executable", name)
		}
	}
	return true, "installed at " + wantHooksPath
}
