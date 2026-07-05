package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/config"
	"github.com/YellowFoxH4XOR/dwarpal/internal/hooks"
)

// starterConfig is written by `dwarpal init`. The compiled-in defaults already
// cover everything here — this file exists only so a new user sees the knobs.
// Dwarpal runs with no config at all if you delete it.
const starterConfig = `version: 1
mode: enforce            # enforce | warn | ci_strict

provenance:
  branch_prefixes: ["agent/", "ai/"]
  trailers: ["Claude", "GitHub Copilot", "Cursor", "Devin", "Aider"]
  apply_gates_to: all-commits  # all-commits (default) | agent-only (exempt humans)

gates:
  diff_budget:
    max_lines: 300
    max_files: 20
    max_new_files: 10
    overrides:
      - paths: ["generated/**", "**/*.lock"]
        max_lines: 10000
  branch_policy:
    protected: ["main", "release/*"]
  ai_patterns:
    enabled: true
    disable_rules: []          # e.g. ["no-broad-catch"]
  scope:
    require_task_manifest: false
    allow_always: ["**/*.lock", "**/__snapshots__/**"]
`

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up Dwarpal in this repo: write config and install git hooks",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInit()
		},
	}
}

func runInit() error {
	if !gitAvailable() {
		return &exitError{code: 2, msg: "system git executable is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required (run 'git init' first)"}
	}

	cfgPath := filepath.Join(root, config.Filename)
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("• %s already exists — leaving it untouched\n", config.Filename)
	} else {
		if err := os.WriteFile(cfgPath, []byte(starterConfig), 0o644); err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
		fmt.Printf("• wrote starter %s\n", config.Filename)
	}

	actions, err := hooks.Install(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	for _, a := range actions {
		fmt.Printf("• %s\n", a)
	}

	fmt.Println("\nDwarpal is guarding the gate. Try: dwarpal check")
	return nil
}
