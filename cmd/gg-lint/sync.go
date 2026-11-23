package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/pkg/harness"
)

// instructions is the gg-lint instruction block synced into every harness's
// instruction file.
const instructions = "Run `gg-lint scan --git-changes` to check changed Go files before completing Go code changes."

// harnessNames returns the name of every known harness, in name order.
func harnessNames() []string {
	all := harness.All()
	names := make([]string, 0, len(all))
	for _, h := range all {
		names = append(names, h.Name())
	}
	return names
}

// newSyncInstructionsCommand builds the sync-instructions command. It
// updates only the named harness's instruction file: the user-level one by
// default or with --user, or the one in the current directory with --local.
// Scanning never calls this operation.
func newSyncInstructionsCommand() *cobra.Command {
	var local, user bool
	names := harnessNames()
	cmd := &cobra.Command{
		Use:       fmt.Sprintf("sync-instructions <%s>", strings.Join(names, "|")),
		Short:     "Keep a coding-agent harness's instructions current.",
		Long:      "Keep a coding-agent harness's instructions current. By default, or with\n--user, the harness's user-level file is updated (~/.claude/CLAUDE.md,\n~/.codex/AGENTS.md); with --local, the file in the current directory is.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: names,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, ok := harness.Lookup(args[0])
			if !ok {
				return fmt.Errorf("unknown harness %q (want one of %s)", args[0], strings.Join(names, ", "))
			}
			scope, dir := "user", "."
			if local {
				scope = "local"
			} else {
				var err error
				if dir, err = h.UserDir(); err != nil {
					return fmt.Errorf("resolve %s user directory: %w", h.Name(), err)
				}
			}
			if abs, err := filepath.Abs(dir); err == nil {
				dir = abs
			}
			changed, err := h.Sync(dir, instructions)
			if err != nil {
				return fmt.Errorf("sync %s %s instructions in %s: %w", h.Name(), scope, dir, err)
			}
			if changed {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated %s %s instructions in %s.\n", h.Name(), scope, dir)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s instructions already current in %s.\n", h.Name(), scope, dir)
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "update the instruction file in the current directory")
	cmd.Flags().BoolVar(&user, "user", false, "update the harness's user-level instruction file (default)")
	cmd.MarkFlagsMutuallyExclusive("local", "user")
	return cmd
}
