package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCommand builds the version command, printing the version,
// commit, and build time baked in at build time.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version information.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "gg-lint %s\ncommit: %s\nbuilt: %s\n", version, commit, builtAt)
			return nil
		},
	}
}
