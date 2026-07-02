package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/taskmanifest"
)

func newTaskCmd() *cobra.Command {
	var paths []string
	cmd := &cobra.Command{
		Use:   "task <id>",
		Short: "Declare the scope of the current task (writes .dwarpal-task.yml)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runTask(args[0], paths)
		},
	}
	cmd.Flags().StringArrayVar(&paths, "paths", nil, "path globs this task is allowed to touch (repeatable)")
	return cmd
}

func runTask(id string, paths []string) error {
	if len(paths) == 0 {
		return &exitError{code: 2, msg: "at least one --paths glob is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}
	if err := taskmanifest.Write(root, id, paths); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	fmt.Printf("• wrote %s for task %q (%d path glob(s))\n", taskmanifest.Filename, id, len(paths))
	return nil
}
