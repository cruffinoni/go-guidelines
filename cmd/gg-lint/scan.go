package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
	"github.com/cruffinoni/go-guidelines/internal/config"
	"github.com/cruffinoni/go-guidelines/internal/engine"
	"github.com/cruffinoni/go-guidelines/internal/lint"
	"github.com/cruffinoni/go-guidelines/internal/loader"
	"github.com/cruffinoni/go-guidelines/internal/scope"
	"github.com/cruffinoni/go-guidelines/pkg/git"
	"github.com/cruffinoni/go-guidelines/pkg/report"
)

// scanFlags holds the raw flag values for one scan invocation.
type scanFlags struct {
	format       string
	failOn       string
	configPath   string
	noConfig     bool
	verbose      bool
	gitChanges   bool
	excludes     []string
	enableRules  []string
	disableRules []string
}

// worktreeRoots returns the distinct, canonical Git worktree roots that
// contain paths. Paths outside any worktree are ignored.
func worktreeRoots(ctx context.Context, paths []string) ([]string, error) {
	seen := make(map[string]bool, len(paths))
	var roots []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", p, err)
		}
		dir := p
		if !info.IsDir() {
			dir = filepath.Dir(p)
		}
		root, err := git.Root(ctx, dir)
		if err != nil {
			if errors.Is(err, git.ErrNotRepository) || errors.Is(err, exec.ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("resolve project root for %s: %w", p, err)
		}
		canonical, err := filepath.EvalSymlinks(root)
		if err != nil {
			canonical = filepath.Clean(root)
		}
		if !seen[canonical] {
			seen[canonical] = true
			roots = append(roots, canonical)
		}
	}
	slices.Sort(roots)
	return roots, nil
}

// resolveConfigPath returns the config path discovered from the Git
// worktrees containing paths, then from the user config directory, or "" if
// neither has a config file. With --no-config it returns "" immediately.
// Multiple distinct worktrees require an explicit --config because there is
// no unambiguous project config to load.
func resolveConfigPath(ctx context.Context, paths []string, explicit string, noConfig bool) (string, error) {
	switch {
	case noConfig:
		return "", nil
	case explicit != "":
		return explicit, nil
	}

	roots, err := worktreeRoots(ctx, paths)
	if err != nil {
		return "", err
	}
	switch len(roots) {
	case 0:
		return config.Discover("")
	case 1:
		return config.Discover(roots[0])
	default:
		return "", fmt.Errorf("scan spans multiple Git worktrees (%s); pass --config or --no-config", strings.Join(roots, ", "))
	}
}

// goBuildExcluded reports whether rel is skipped by go build ./...
// semantics: testdata and vendor directories, dot/underscore-prefixed
// directories, and dot/underscore-prefixed files.
func goBuildExcluded(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." {
			continue
		}
		if part == "testdata" || part == "vendor" {
			return true
		}
		if strings.HasPrefix(part, ".") || strings.HasPrefix(part, "_") {
			return true
		}
	}
	return false
}

// directoryScanAllows reports whether name is reachable under one of dirs
// without crossing a directory that go build ./... would skip.
func directoryScanAllows(name string, dirs []string) bool {
	for _, dir := range dirs {
		rel, err := filepath.Rel(dir, name)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			continue
		}
		if goBuildExcluded(rel) {
			continue
		}
		if targetModule, ok := loader.ModuleRoot(dir); ok {
			if changedModule, ok := loader.ModuleRoot(filepath.Dir(name)); ok && changedModule != targetModule {
				continue
			}
		}
		return true
	}
	return false
}

// filterChangedFiles keeps the changed Go files that a directory scan of
// targets would analyze. It excludes testdata, vendor, hidden/underscore
// directories, underscore/dot-prefixed files, and files owned by a nested
// module. Explicit file targets are always kept.
func filterChangedFiles(changed []string, targets *scope.Set) []string {
	dirs := targets.DirectoryInputs()
	var kept []string
	for _, name := range changed {
		if !strings.HasSuffix(name, ".go") || !targets.Contains(name) {
			continue
		}
		if targets.IsFileInput(name) {
			kept = append(kept, name)
			continue
		}
		if directoryScanAllows(name, dirs) {
			kept = append(kept, name)
		}
	}
	return kept
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

// buildScope resolves the scan's scope.Set from paths and the exclude
// flag. With --git-changes, it narrows paths to the Go files changed in
// the Git worktrees containing them that a directory scan would analyze,
// returning an empty Set when nothing remains.
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

	kept := filterChangedFiles(changed, targets)
	if len(kept) == 0 {
		return scope.NewEmpty(), nil
	}
	return scope.New(kept, f.excludes)
}

// trimRecursiveSuffix strips a trailing "/..." or "\..." from every path,
// as in "./pkg/...": directory inputs are always scanned recursively, so
// the suffix is accepted for familiarity with go vet and go test.
func trimRecursiveSuffix(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		switch {
		case p == "...":
			p = "."
		case strings.HasSuffix(p, "...") && len(p) > 3 && (p[len(p)-4] == '/' || p[len(p)-4] == '\\'):
			p = p[:len(p)-len("...")]
		}
		out[i] = p
	}
	return out
}

// reportPath returns file relative to cwd when the two are on the same
// volume, and an absolute slash-separated path otherwise (as on Windows).
func reportPath(cwd, file string) string {
	rel, err := filepath.Rel(cwd, file)
	if err != nil {
		rel = file
	}
	return filepath.ToSlash(rel)
}

// toReport converts findings to a report.Report, with every finding's
// File made relative to the working directory, slash-separated, as
// go vet does. When the paths are on different volumes (Windows), the
// absolute slash-separated path is used instead.
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
		out[i] = report.Finding{
			RuleID:   f.RuleID,
			Severity: f.Severity.String(),
			Message:  f.Message,
			File:     reportPath(cwd, f.File),
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

	configPath, err := resolveConfigPath(ctx, paths, flags.configPath, flags.noConfig)
	if err != nil {
		return err
	}
	if flags.verbose {
		if configPath == "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "gg-lint: config: none")
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "gg-lint: config: %s\n", configPath)
		}
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

	tests := true
	if !set.Empty() && (flags.gitChanges || !set.HasDirectoryInput()) {
		tests = false
		for _, p := range set.Paths() {
			if strings.HasSuffix(p, "_test.go") {
				tests = true
				break
			}
		}
	}
	eng := engine.New(cat, engine.Options{
		Rules: func(m lint.Meta) bool { return !opts.disabled[m.ID] },
		Tests: &tests,
	})
	res, err := eng.Check(ctx, set)
	if err != nil {
		return err
	}
	for _, typeErr := range res.TypeErrors {
		fmt.Fprintf(cmd.ErrOrStderr(), "gg-lint: warning: %v\n", typeErr)
	}
	if len(res.Files) == 0 {
		if !flags.gitChanges {
			return errors.New("no Go source files found")
		}
		// A changed-file scan whose only changes are generated,
		// build-tag-excluded, or otherwise not lintable is a successful
		// no-op rather than an operational failure.
		if err := format.Write(cmd.OutOrStdout(), report.Report{}); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		return nil
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
	fs.StringVar(&flags.configPath, "config", "", "path to configuration YAML (default: <target-git-root>/gg-lint.yaml, then ~/.config/gg-lint/config.yaml)")
	fs.BoolVar(&flags.noConfig, "no-config", false, "ignore project and user configuration files")
	fs.BoolVar(&flags.verbose, "verbose", false, "print resolved configuration and scan diagnostics to stderr")
	fs.BoolVar(&flags.gitChanges, "git-changes", false, "scan only Go files changed in the Git worktree (vs HEAD, plus untracked)")
	fs.StringArrayVar(&flags.excludes, "exclude", nil, "exclude a path (repeatable)")
	fs.StringArrayVar(&flags.enableRules, "enable-rule", nil, "enable a rule (repeatable)")
	fs.StringArrayVar(&flags.disableRules, "disable-rule", nil, "disable a rule (repeatable)")
	cmd.MarkFlagsMutuallyExclusive("config", "no-config")
	return cmd
}
