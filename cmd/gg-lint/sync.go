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

// runHarnessAction resolves the selected harness and scope before editing.
func runHarnessAction(cmd *cobra.Command, name string, local, remove bool) error {
	h, ok := harness.Lookup(name)
	if !ok {
		return fmt.Errorf("unknown harness %q (run %s --help for the list)", name, cmd.CommandPath())
	}
	scope, other := harness.User, "--local"
	var projectDir string
	if local {
		scope, other = harness.Local, "--user"
		// User scope ignores projectDir and does not need Git.
		dir, err := projectRoot(cmd.Context())
		if err != nil {
			return err
		}
		projectDir = dir
	}

	var (
		path    string
		changed bool
		err     error
	)
	verb := "set up"
	if remove {
		verb = "remove"
		path, changed, err = h.Remove(scope, projectDir)
	} else {
		path, changed, err = h.Sync(scope, projectDir, instructions)
	}
	switch {
	case errors.Is(err, harness.ErrUnsupportedScope):
		return fmt.Errorf("%w; use %s", err, other)
	case errors.Is(err, harness.ErrMissingDirectory) && !remove:
		return fmt.Errorf("%s %s %s instructions: %w; create that directory first or use %s", verb, h.Name(), scope, err, other)
	case err != nil:
		return fmt.Errorf("%s %s %s instructions: %w", verb, h.Name(), scope, err)
	}
	switch {
	case remove && changed:
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %s %s instructions: %s\n", h.Name(), scope, path)
	case remove:
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "No managed %s %s instructions to remove: %s\n", h.Name(), scope, path)
	case changed:
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated %s %s instructions: %s\n", h.Name(), scope, path)
	default:
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s instructions already current: %s\n", h.Name(), scope, path)
	}
	return err
}

// newHarnessActionCommand builds setup, remove, or the legacy spelling of setup.
func newHarnessActionCommand(name string, remove bool) *cobra.Command {
	var local bool
	short := "Add or update the managed gg-lint instructions."
	details := "For setup, the top-level user directory must already exist."
	if remove {
		short = "Remove the managed gg-lint instructions without deleting the file."
		details = "A missing file or block is a no-op; no files or directories are created."
	}
	names := harnessNames()
	cmd := &cobra.Command{
		Use:   name + " <harness>",
		Short: short,
		Long: short + "\n\n" +
			"By default, or with --user, the operation targets the harness's user-level\n" +
			"file (for example ~/.claude/CLAUDE.md). With --local, it targets the\n" +
			"project-level file at the Git repository root, or in the current directory\n" +
			"outside a repository. Harnesses that share a file share one gg-lint block.\n" +
			details + "\n\nHarnesses: " + strings.Join(names, ", ") + ".",
		Args:      cobra.ExactArgs(1),
		ValidArgs: names,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHarnessAction(cmd, args[0], local, remove)
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "select the project-level instruction file at the repository root")
	cmd.Flags().Bool("user", false, "select the harness's user-level instruction file (default)")
	cmd.MarkFlagsMutuallyExclusive("local", "user")
	return cmd
}

// newHarnessCommand groups the coding-agent instruction commands.
func newHarnessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "harness",
		Short: "Manage gg-lint instructions for coding-agent harnesses.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newHarnessActionCommand("setup", false), newHarnessActionCommand("remove", true))
	return cmd
}

// newLegacySyncCommand keeps existing sync-instructions invocations working.
func newLegacySyncCommand() *cobra.Command {
	cmd := newHarnessActionCommand("sync-instructions", false)
	cmd.Hidden = true
	run := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.PrintErrln("gg-lint: sync-instructions is deprecated; use gg-lint harness setup")
		return run(cmd, args)
	}
	return cmd
}
