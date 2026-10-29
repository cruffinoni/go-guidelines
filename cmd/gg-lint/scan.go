package main

import (
	"context"
	"errors"
	"fmt"
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
	gitChanges   bool
	excludes     []string
	enableRules  []string
	disableRules []string
}

// resolveConfigPath returns the config path discovered from the Git
// worktree root containing the working directory, then from the user
// config directory, or "" if neither has a config file. Outside a Git
// worktree only the user config directory is searched.
func resolveConfigPath(ctx context.Context) (string, error) {
	root, err := git.Root(ctx, ".")
	if err != nil {
		if !errors.Is(err, git.ErrNotRepository) {
			return "", fmt.Errorf("resolve project root: %w", err)
		}
		root = ""
	}
	return config.Discover(root)
}

// buildScope resolves the scan's scope.Set from paths and the exclude
// flag. With --git-changes, it narrows paths to the Go files changed in
// the Git worktrees containing them, failing if none remain.
func (f scanFlags) buildScope(ctx context.Context, paths []string) (*scope.Set, error) {
	if !f.gitChanges {
		return scope.New(paths, f.excludes)
	}

	targets, err := scope.New(paths, nil)
	if err != nil {
		return nil, err
	}
	changed, err := changedFiles(ctx, paths)
	if err != nil {
		return nil, err
	}

	var kept []string
	for _, name := range changed {
		if strings.HasSuffix(name, ".go") && targets.Contains(name) {
			kept = append(kept, name)
		}
	}
	if len(kept) == 0 {
		return nil, errors.New("no changed Go source files found")
	}
	return scope.New(kept, f.excludes)
}

// changedFiles returns the files changed in every Git worktree containing
// one of paths, each worktree listed once. Unlike config discovery, it
// fails when a path lies outside any worktree.
func changedFiles(ctx context.Context, paths []string) ([]string, error) {
	seen := make(map[string]bool)
	var changed []string
	for _, p := range paths {
		dir := p
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			dir = filepath.Dir(p)
		}
		root, err := git.Root(ctx, dir)
		if err != nil {
			return nil, fmt.Errorf("resolve changed files in %s: %w", p, err)
		}
		if seen[root] {
			continue
		}
		seen[root] = true

		files, err := git.ChangedFiles(ctx, root)
		if err != nil {
			return nil, fmt.Errorf("list changed files in %s: %w", root, err)
		}
		changed = append(changed, files...)
	}
	return changed, nil
}

// trimRecursiveSuffix strips a trailing "/..." from every path, as in
// "./pkg/...": directory inputs are always scanned recursively, so the
// suffix is accepted for familiarity with go vet and go test.
func trimRecursiveSuffix(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		switch {
		case p == "...":
			p = "."
		case strings.HasSuffix(p, "/..."):
			p = strings.TrimSuffix(p, "...")
		}
		out[i] = p
	}
	return out
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

// runScan resolves configuration, builds the scope, runs the active rules,
// and writes the report to cmd's output. It returns errThreshold when
// findings reach --fail-on, and any other error for a usage or operational
// failure.
func runScan(cmd *cobra.Command, cat *catalog.Catalog, paths []string, flags scanFlags) error {
	ctx := cmd.Context()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	paths = trimRecursiveSuffix(paths)

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

	set, err := flags.buildScope(ctx, paths)
	if err != nil {
		return err
	}

	eng := engine.New(cat, engine.Options{Rules: func(m lint.Meta) bool { return !opts.disabled[m.ID] }})
	res, err := eng.Check(ctx, set)
	if err != nil {
		return err
	}
	for _, typeErr := range res.TypeErrors {
		fmt.Fprintf(cmd.ErrOrStderr(), "gg-lint: warning: %v\n", typeErr)
	}
	if len(res.Files) == 0 {
		return errors.New("no Go source files found")
	}

	rep, err := toReport(res.Findings, len(res.Files))
	if err != nil {
		return err
	}
	if err := format.Write(cmd.OutOrStdout(), rep); err != nil {
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

// newScanCommand builds the scan command over cat. Every flag defaults to
// the built-in default; the config file, then flags, override it, checked
// through cmd.Flags().Changed so an unset flag never masks a config value.
func newScanCommand(cat *catalog.Catalog) *cobra.Command {
	var flags scanFlags
	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Long:  "Scan Go source files against the rule catalog. Directory paths are scanned\nrecursively; a trailing /... is accepted and changes nothing.",
		Short: "Scan Go source files against the rule catalog.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, cat, args, flags)
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&flags.format, "format", "text", "report format: text or json")
	fs.StringVar(&flags.failOn, "fail-on", "error", "minimum severity that causes exit code 1: none, info, warning, or error")
	fs.StringVar(&flags.configPath, "config", "", "path to configuration YAML (default: <git-root>/gg-lint.yaml, then ~/.config/gg-lint/config.yaml)")
	fs.BoolVar(&flags.gitChanges, "git-changes", false, "scan only Go files changed in the Git worktree (vs HEAD, plus untracked)")
	fs.StringArrayVar(&flags.excludes, "exclude", nil, "exclude a path (repeatable)")
	fs.StringArrayVar(&flags.enableRules, "enable-rule", nil, "enable a rule (repeatable)")
	fs.StringArrayVar(&flags.disableRules, "disable-rule", nil, "disable a rule (repeatable)")
	return cmd
}
