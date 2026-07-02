package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("dwarpal %s (commit %s, built %s)\n", version, commit, date)
			return nil
		},
	}
}
