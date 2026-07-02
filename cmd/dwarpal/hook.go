package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/hooks"
)

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage Dwarpal's git hooks",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "install",
			Short: "Install pre-commit and pre-push hooks (chains to existing hooks)",
			RunE:  func(_ *cobra.Command, _ []string) error { return runHook(hooks.Install) },
		},
		&cobra.Command{
			Use:   "uninstall",
			Short: "Remove Dwarpal's hooks and restore prior hook configuration",
			RunE:  func(_ *cobra.Command, _ []string) error { return runHook(hooks.Uninstall) },
		},
	)
	return cmd
}

func runHook(fn func(string) ([]string, error)) error {
	if !gitAvailable() {
		return &exitError{code: 2, msg: "system git executable is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}
	actions, err := fn(root)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	for _, a := range actions {
		fmt.Printf("• %s\n", a)
	}
	return nil
}
