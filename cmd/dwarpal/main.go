// Command dwarpal is the CLI entrypoint — the quality firewall between AI
// coding agents and your repository.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build metadata, injected via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// exitError carries a specific process exit code up to main. Exit codes are a
// contract (PRD §5.4): 0 pass, 1 blocked, 2 config/internal error.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func main() { os.Exit(run()) }

// run executes the CLI and returns the process exit code. Splitting this from
// main (which only calls os.Exit) gives tests a seam to drive the CLI without
// terminating the test process.
func run() int {
	root := &cobra.Command{
		Use:           "dwarpal",
		Short:         "A quality firewall between AI coding agents and your repository",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(
		newCheckCmd(), newInitCmd(), newHookCmd(), newRulesCmd(), newTaskCmd(),
		newExplainCmd(), newDoctorCmd(), newBypassCmd(), newFeedbackCmd(), newAgentCmd(), newAnalyzeCmd(), newVersionCmd(),
	)

	if err := root.Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			if ee.msg != "" {
				fmt.Fprintln(os.Stderr, "dwarpal: "+ee.msg)
			}
			return ee.code
		}
		fmt.Fprintln(os.Stderr, "dwarpal: "+err.Error())
		return 2
	}
	return 0
}
