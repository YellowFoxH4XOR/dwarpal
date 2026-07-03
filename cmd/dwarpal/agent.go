package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/agentsetup"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Wire Dwarpal into a coding agent's workflow",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "setup <" + strings.Join(agentsetup.SupportedTools(), "|") + ">",
		Short: "Teach an agent this repo's gate: instruction block + skill + (Claude Code) pre-flight hook",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAgentSetup(args[0])
		},
	})
	return cmd
}

func runAgentSetup(toolArg string) error {
	tool := agentsetup.Tool(toolArg)
	supported := false
	for _, t := range agentsetup.SupportedTools() {
		if toolArg == t {
			supported = true
		}
	}
	if !supported {
		return &exitError{code: 2, msg: fmt.Sprintf("unknown tool %q (supported: %s)", toolArg, strings.Join(agentsetup.SupportedTools(), ", "))}
	}

	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}

	path, created, err := agentsetup.UpsertInstructions(root, tool)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	verb := "updated dwarpal block in"
	if created {
		verb = "created"
	}
	fmt.Printf("• %s %s\n", verb, path)

	skillPath, skillCreated, err := agentsetup.UpsertSkill(root, tool)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	skillVerb := "refreshed"
	if skillCreated {
		skillVerb = "installed"
	}
	fmt.Printf("• %s Dwarpal skill at %s\n", skillVerb, skillPath)

	if tool == agentsetup.ToolClaudeCode {
		hookPath, added, err := agentsetup.MergeClaudeSettings(root)
		if err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
		if added {
			fmt.Printf("• added pre-flight PreToolUse hook to %s\n", hookPath)
		} else {
			fmt.Printf("• pre-flight hook already present in %s\n", hookPath)
		}
	}

	fmt.Printf("\n%s now pre-flights the gate before committing. Pair with `dwarpal init` for the enforcement hooks.\n", toolArg)
	return nil
}
