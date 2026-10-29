package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/pkg/git"
	"github.com/cruffinoni/go-guidelines/pkg/harness"
)

// instructions is the gg-lint instruction block synced into every harness's
// project instruction file.
const instructions = "Run `gg-lint scan --git-changes` to check changed Go files before completing Go code changes."

// harnessNames joins the names of every known harness for use in flag
// usage and error messages.
func harnessNames() string {
	all := harness.All()
	names := make([]string, 0, len(all))
	for _, h := range all {
		names = append(names, h.Name())
	}
	return strings.Join(names, ", ")
}

// newSyncInstructionsCommand builds the sync-instructions command. It
// updates only the selected harness's project instruction file; scanning
// never calls this operation.
func newSyncInstructionsCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "sync-instructions",
		Short: "Keep a coding-agent harness's project instructions current.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, ok := harness.Lookup(name)
			if !ok {
				return fmt.Errorf("unknown harness %q (want one of %s)", name, harnessNames())
			}
			root, err := git.Root(cmd.Context(), ".")
			if err != nil {
				return fmt.Errorf("resolve project root: %w", err)
			}
			changed, err := h.Sync(root, instructions)
			if err != nil {
				return err
			}
			if changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated %s project instructions.\n", h.Name())
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Instructions already current (%s).\n", h.Name())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "harness", "", fmt.Sprintf("coding-agent harness to update: %s (required)", harnessNames()))
	return cmd
}
