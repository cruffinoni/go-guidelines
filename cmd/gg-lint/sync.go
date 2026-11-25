package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/pkg/git"
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

// projectRoot returns the root of the Git repository containing the current
// directory, so --local writes where the agent reads project instructions,
// or the current directory itself outside a repository.
func projectRoot(ctx context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	root, err := git.Root(ctx, cwd)
	switch {
	case errors.Is(err, git.ErrNotRepository):
		return cwd, nil
	case err != nil:
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	return root, nil
}

// newSyncInstructionsCommand builds the sync-instructions command. It
// updates only the named harness's instruction file: the user-level one by
// default or with --user, or the project-level one with --local.
// Scanning never calls this operation.
func newSyncInstructionsCommand() *cobra.Command {
	var local, user bool
	names := harnessNames()
	cmd := &cobra.Command{
		Use:   "sync-instructions <harness>",
		Short: "Keep a coding-agent harness's instructions current.",
		Long: "Keep a coding-agent harness's instructions current. By default, or with\n" +
			"--user, the harness's user-level file is updated (for example\n" +
			"~/.claude/CLAUDE.md); its directory must already exist. With --local, the\n" +
			"project-level file at the Git repository root is updated (for example\n" +
			"./CLAUDE.md), or under the current directory outside a repository. Harnesses that share a file share one gg-lint block.\n\n" +
			"Harnesses: " + strings.Join(names, ", ") + ".",
		Args:      cobra.ExactArgs(1),
		ValidArgs: names,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, ok := harness.Lookup(args[0])
			if !ok {
				return fmt.Errorf("unknown harness %q (run gg-lint sync-instructions --help for the list)", args[0])
			}
			scope, other := harness.User, "--local"
			if local {
				scope, other = harness.Local, "--user"
			}
			projectDir, err := projectRoot(cmd.Context())
			if err != nil {
				return err
			}
			path, changed, err := h.Sync(scope, projectDir, instructions)
			switch {
			case errors.Is(err, harness.ErrUnsupportedScope):
				return fmt.Errorf("%w; use %s", err, other)
			case err != nil:
				return fmt.Errorf("sync %s %s instructions: %w", h.Name(), scope, err)
			case changed:
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated %s %s instructions: %s\n", h.Name(), scope, path)
			default:
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s instructions already current: %s\n", h.Name(), scope, path)
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "update the project-level instruction file at the repository root")
	cmd.Flags().BoolVar(&user, "user", false, "update the harness's user-level instruction file (default)")
	cmd.MarkFlagsMutuallyExclusive("local", "user")
	return cmd
}
