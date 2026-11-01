package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
)

// errThreshold signals that scan found findings at or above --fail-on. run
// maps it to exit code 1 without printing an additional error: the report
// itself, already written to stdout, explains the failure.
var errThreshold = errors.New("findings at or above the fail-on threshold")

// exitCode maps err to a process exit code, printing operational errors
// once on stderr.
func exitCode(stderr io.Writer, err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errThreshold):
		return 1
	default:
		fmt.Fprintf(stderr, "gg-lint: %v\n", err)
		return 2
	}
}

// newRootCommand builds the gg-lint command tree over cat, writing to
// stdout and stderr; subcommands reach them through cmd.OutOrStdout and
// cmd.ErrOrStderr. Subcommand errors are neither auto-printed nor
// auto-reported as usage, since run reports them once after mapping them
// to an exit code.
func newRootCommand(stdout, stderr io.Writer, cat *catalog.Catalog) *cobra.Command {
	root := &cobra.Command{
		Use:           "gg-lint",
		Short:         "gg-lint lints Go code against the Go guidelines.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(
		newScanCommand(cat),
		newRulesCommand(cat),
		newSyncInstructionsCommand(),
		newVersionCommand(),
	)
	return root
}

// run builds the gg-lint command tree over cat, executes args against it,
// and maps the result to a process exit code: 0 for a clean run, 1 when
// scan finds findings at or above --fail-on, 2 for a usage or operational
// error.
func run(ctx context.Context, args []string, stdout, stderr io.Writer, cat *catalog.Catalog) int {
	root := newRootCommand(stdout, stderr, cat)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	return exitCode(stderr, err)
}
