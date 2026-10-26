package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
	"github.com/cruffinoni/go-guidelines/internal/config"
	"github.com/cruffinoni/go-guidelines/internal/engine"
	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/scope"
	"github.com/cruffinoni/go-guidelines/pkg/git"
	"github.com/cruffinoni/go-guidelines/pkg/report"
)

// scanFlags holds the raw flag values for one scan invocation.
type scanFlags struct {
	format       string
	failOn       string
	configPath   string
	changedFiles bool
	includes     []string
	excludes     []string
	enableRules  []string
	disableRules []string
}

// newScanCommand builds the scan command over cat. Every flag defaults to
// the built-in default; the config file, then flags, override it, checked
// through cmd.Flags().Changed so an unset flag never masks a config value.
func newScanCommand(stdout, stderr io.Writer, cat *catalog.Catalog) *cobra.Command {
	var flags scanFlags
	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Scan Go source files against the rule catalog.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, stdout, stderr, cat, args, flags)
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&flags.format, "format", "text", "report format: text or json")
	fs.StringVar(&flags.failOn, "fail-on", "error", "minimum severity that causes exit code 1: none, info, warning, or error")
	fs.StringVar(&flags.configPath, "config", "", "path to project configuration TOML")
	fs.BoolVar(&flags.changedFiles, "changed-files", false, "scan changed Go files in the Git worktree")
	fs.StringArrayVar(&flags.includes, "include", nil, "include a path (repeatable)")
	fs.StringArrayVar(&flags.excludes, "exclude", nil, "exclude a path (repeatable)")
	fs.StringArrayVar(&flags.enableRules, "enable-rule", nil, "enable a rule (repeatable)")
	fs.StringArrayVar(&flags.disableRules, "disable-rule", nil, "disable a rule (repeatable)")
	return cmd
}

// runScan resolves configuration, builds the scope, runs the active rules,
// and writes the report. It returns errThreshold when findings reach
// --fail-on, and any other error for a usage or operational failure.
func runScan(cmd *cobra.Command, stdout, stderr io.Writer, cat *catalog.Catalog, paths []string, flags scanFlags) error {
	ctx := cmd.Context()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	configPath := flags.configPath
	if configPath == "" {
		resolved, err := resolveConfigPath(ctx)
		if err != nil {
			return err
		}
		configPath = resolved
	}

	opts, err := resolveOptions(cat, cmd, flags, configPath)
	if err != nil {
		return err
	}

	format, err := report.ParseFormat(opts.format)
	if err != nil {
		return err
	}
	threshold, ok := parseFailOn(opts.failOn)
	if !ok {
		return fmt.Errorf("invalid --fail-on value %q", opts.failOn)
	}

	set, err := buildScope(ctx, paths, flags.includes, flags.excludes, flags.changedFiles)
	if err != nil {
		return err
	}

	eng := engine.New(cat, engine.Options{Rules: func(m lint.Meta) bool { return !opts.disabled[m.ID] }})
	res, err := eng.Check(ctx, set)
	if err != nil {
		return err
	}
	for _, typeErr := range res.TypeErrors {
		fmt.Fprintf(stderr, "gg-lint: warning: %v\n", typeErr)
	}
	if len(res.Files) == 0 {
		return errors.New("no Go source files found")
	}

	rep, err := toReport(res.Findings, len(res.Files))
	if err != nil {
		return err
	}
	if err := format.Write(stdout, rep); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	if threshold != 0 {
		for _, f := range res.Findings {
			if f.Severity >= threshold {
				return errThreshold
			}
		}
	}
	return nil
}

// resolveConfigPath returns the project config path discovered from the
// Git worktree root containing the working directory, or "" if the
// working directory is not inside a Git worktree or has no config file.
func resolveConfigPath(ctx context.Context) (string, error) {
	root, err := git.Root(ctx, ".")
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			return "", nil
		}
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	return config.Discover(root)
}

// buildScope resolves the scan's scope.Set from paths, include, and
// exclude. With changedFiles, it narrows paths to the Go files changed in
// the Git worktree that also lie within paths, failing if none remain.
func buildScope(ctx context.Context, paths, include, exclude []string, changedFiles bool) (*scope.Set, error) {
	if !changedFiles {
		return scope.New(paths, include, exclude)
	}

	// Unlike config discovery, --changed-files cannot work outside a worktree.
	root, err := git.Root(ctx, ".")
	if err != nil {
		return nil, fmt.Errorf("resolve changed files: %w", err)
	}
	targets, err := scope.New(paths, nil, nil)
	if err != nil {
		return nil, err
	}
	changed, err := git.ChangedFiles(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("list changed files: %w", err)
	}

	var kept []string
	for _, f := range changed {
		if strings.HasSuffix(f, ".go") && targets.Contains(f) {
			kept = append(kept, f)
		}
	}
	if len(kept) == 0 {
		return nil, errors.New("no changed Go source files found")
	}
	return scope.New(kept, include, exclude)
}

// toReport converts findings to a report.Report, with every finding's
// File made relative to the working directory, slash-separated, as
// go vet does.
func toReport(findings []lint.Finding, scannedFiles int) (report.Report, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return report.Report{}, fmt.Errorf("get working directory: %w", err)
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return report.Report{}, fmt.Errorf("resolve working directory: %w", err)
	}

	out := make([]report.Finding, len(findings))
	for i, f := range findings {
		rel, err := filepath.Rel(cwd, f.File)
		if err != nil {
			return report.Report{}, fmt.Errorf("relativize %s: %w", f.File, err)
		}
		out[i] = report.Finding{
			RuleID:   f.RuleID,
			Severity: f.Severity.String(),
			Message:  f.Message,
			File:     filepath.ToSlash(rel),
			Line:     f.Line,
			Column:   f.Column,
		}
	}
	return report.Report{ScannedFiles: scannedFiles, Findings: out}, nil
}
