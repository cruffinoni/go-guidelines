package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/cruffinoni/go-guidelines/internal/fixes"
	"github.com/cruffinoni/go-guidelines/internal/rules"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const reportSchemaVersion = "1.0"

type report struct {
	SchemaVersion string          `json:"schema_version"`
	ScannedFiles  int             `json:"scanned_files"`
	Findings      []rules.Finding `json:"findings"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "", "report format: text or json")
	failOn := flags.String("fail-on", "", "minimum severity that causes exit code 1: error, warning, info, or none")
	configPath := flags.String("config", "", "path to project configuration TOML")
	changedFiles := flags.Bool("changed-files", false, "scan changed Go files in the Git worktree")
	applyFixes := flags.Bool("fix", false, "apply supported in-process fixes (explicit opt-in)")
	previewFixes := flags.Bool("fix-dry-run", false, "preview supported in-process fixes without changing files")
	var includes, excludes, enabled, disabled stringList
	flags.Var(&includes, "include", "include a path (repeatable)")
	flags.Var(&excludes, "exclude", "exclude a path (repeatable)")
	flags.Var(&enabled, "enable-rule", "enable a rule (repeatable)")
	flags.Var(&disabled, "disable-rule", "disable a rule (repeatable)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *applyFixes && *previewFixes {
		fmt.Fprintln(stderr, "gg-lint: --fix and --fix-dry-run cannot be used together")
		return 2
	}
	rulesByID := make(map[string]rules.Rule)
	for _, rule := range rules.BuiltInRules() {
		rulesByID[rule.ID()] = rule
	}
	knownRules := make(map[string]struct{}, len(rulesByID))
	for id := range rulesByID {
		knownRules[id] = struct{}{}
	}
	resolvedConfigPath := *configPath
	var err error
	if resolvedConfigPath == "" {
		resolvedConfigPath, err = discoverProjectConfig()
		if err != nil {
			fmt.Fprintf(stderr, "gg-lint: discover project config: %v\n", err)
			return 2
		}
	}
	config, err := loadConfig(resolvedConfigPath, knownRules)
	if err != nil {
		fmt.Fprintf(stderr, "gg-lint: %v\n", err)
		return 2
	}
	if isFlagProvided(args, "format") {
		config.format = *format
	}
	if config.format != "text" && config.format != "json" {
		fmt.Fprintf(stderr, "gg-lint: unsupported format %q (want text or json)\n", config.format)
		return 2
	}
	if isFlagProvided(args, "fail-on") {
		config.failOn = *failOn
	}
	threshold, ok := severityRank(config.failOn)
	if !ok {
		fmt.Fprintf(stderr, "gg-lint: invalid --fail-on value %q\n", config.failOn)
		return 2
	}
	active := make(map[string]bool, len(config.rules))
	for id, enabled := range config.rules {
		active[id] = enabled
	}
	flagRules := make(map[string]bool, len(enabled)+len(disabled))
	for _, id := range enabled {
		if _, exists := rulesByID[id]; !exists {
			fmt.Fprintf(stderr, "gg-lint: unknown rule %q\n", id)
			return 2
		}
		if prior, exists := flagRules[id]; exists && !prior {
			fmt.Fprintf(stderr, "gg-lint: conflicting enable/disable flags for rule %q\n", id)
			return 2
		}
		flagRules[id] = true
	}
	for _, id := range disabled {
		if _, exists := rulesByID[id]; !exists {
			fmt.Fprintf(stderr, "gg-lint: unknown rule %q\n", id)
			return 2
		}
		if prior, exists := flagRules[id]; exists && prior {
			fmt.Fprintf(stderr, "gg-lint: conflicting enable/disable flags for rule %q\n", id)
			return 2
		}
		flagRules[id] = false
	}
	for id, enabled := range flagRules {
		active[id] = enabled
	}
	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	changedRoot := ""
	if *changedFiles {
		changed, root, e := gitChangedFiles()
		if e != nil {
			fmt.Fprintf(stderr, "gg-lint: %v\n", e)
			return 2
		}
		changedRoot = root
		paths, e = selectChangedPaths(changed, root, paths, includes, excludes)
		if e != nil {
			fmt.Fprintf(stderr, "gg-lint: %v\n", e)
			return 2
		}
	}
	files, displayPaths, err := discover(paths)
	if err != nil {
		fmt.Fprintf(stderr, "gg-lint: %v\n", err)
		return 2
	}
	if changedRoot != "" {
		for _, filename := range files {
			relative, relErr := filepath.Rel(changedRoot, filename)
			if relErr != nil {
				fmt.Fprintf(stderr, "gg-lint: %v\n", relErr)
				return 2
			}
			displayPaths[filename] = filepath.ToSlash(relative)
		}
	}
	findings := make([]rules.Finding, 0)
	actionOutput := stdout
	if config.format == "json" {
		actionOutput = stderr
	}
	for _, filename := range files {
		source, readErr := os.ReadFile(filename)
		if readErr != nil {
			fmt.Fprintf(stderr, "gg-lint: read %s: %v\n", displayPaths[filename], readErr)
			return 2
		}
		if (*applyFixes || *previewFixes) && active["GBP017"] {
			edits, planErr := fixes.Plan(source)
			if planErr != nil {
				fmt.Fprintf(stderr, "gg-lint: plan fixes for %s: %v\n", displayPaths[filename], planErr)
				return 2
			}
			if len(edits) > 0 {
				updated, applyErr := fixes.Apply(source, edits)
				if applyErr != nil {
					fmt.Fprintf(stderr, "gg-lint: apply fixes for %s: %v\n", displayPaths[filename], applyErr)
					return 2
				}
				if *previewFixes {
					fmt.Fprintf(actionOutput, "Fix preview for %s (no files changed):\n%s", displayPaths[filename], updated)
				} else {
					if writeErr := fixes.WriteAtomic(filename, source, updated); writeErr != nil {
						fmt.Fprintf(stderr, "gg-lint: write fixed source %s: %v\n", displayPaths[filename], writeErr)
						return 2
					}
					fmt.Fprintf(actionOutput, "Applied %d fix(es) to %s.\n", len(edits), displayPaths[filename])
					source = updated
				}
			}
		}
		for id, rule := range rulesByID {
			if !active[id] {
				continue
			}
			found, analyzeErr := rule.Analyze(displayPaths[filename], source)
			if analyzeErr != nil {
				fmt.Fprintf(stderr, "gg-lint: %v\n", analyzeErr)
				return 2
			}
			findings = append(findings, found...)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.RuleID < b.RuleID
	})
	result := report{SchemaVersion: reportSchemaVersion, ScannedFiles: len(files), Findings: findings}
	if config.format == "json" {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintf(stderr, "gg-lint: write JSON report: %v\n", err)
			return 2
		}
	} else {
		for _, finding := range findings {
			fmt.Fprintf(stdout, "%s:%d:%d: %s [%s] %s\n", finding.File, finding.Line, finding.Column, finding.Severity, finding.RuleID, finding.Message)
		}
		if len(findings) == 0 {
			fmt.Fprintf(stdout, "No findings (%d Go files scanned).\n", len(files))
		}
	}
	for _, finding := range findings {
		if threshold > 0 && severityRankValue(finding.Severity) >= threshold {
			return 1
		}
	}
	return 0
}

func isFlagProvided(args []string, name string) bool {
	long, short := "--"+name, "-"+name
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == long || arg == short || strings.HasPrefix(arg, long+"=") || strings.HasPrefix(arg, short+"=") {
			return true
		}
	}
	return false
}

type stringList []string

func (s *stringList) String() string         { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error { *s = append(*s, value); return nil }
func severityRank(value string) (int, bool) {
	rank, ok := map[string]int{"none": -1, "info": 1, "warning": 2, "error": 3}[value]
	return rank, ok
}
func severityRankValue(value rules.Severity) int { rank, _ := severityRank(string(value)); return rank }

func discover(inputs []string) ([]string, map[string]string, error) {
	files := make([]string, 0)
	display := make(map[string]string)
	seen := make(map[string]bool)
	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil {
			return nil, nil, fmt.Errorf("inspect %s: %w", input, err)
		}
		if !info.IsDir() {
			if filepath.Ext(input) != ".go" {
				return nil, nil, fmt.Errorf("%s is not a Go source file or directory", input)
			}
			absolute, err := filepath.Abs(input)
			if err != nil {
				return nil, nil, err
			}
			if !seen[absolute] {
				seen[absolute] = true
				files = append(files, absolute)
				display[absolute] = filepath.Base(input)
			}
			continue
		}
		root, err := filepath.Abs(input)
		if err != nil {
			return nil, nil, err
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && path != root && (entry.Name() == "vendor" || strings.HasPrefix(entry.Name(), ".")) {
				return filepath.SkipDir
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
				rel, e := filepath.Rel(root, path)
				if e != nil {
					return e
				}
				display[path] = filepath.ToSlash(rel)
			}
			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("walk %s: %w", input, err)
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, nil, errors.New("no Go source files found")
	}
	return files, display, nil
}

func gitChangedFiles() ([]string, string, error) {
	rootOutput, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, "", errors.New("--changed-files requires a Git worktree")
	}
	root := strings.TrimSpace(string(rootOutput))
	trackedArgs := []string{"diff", "--name-only", "--diff-filter=ACMRTUXB", "-z", "HEAD"}
	if _, err := exec.Command("git", "rev-parse", "--verify", "HEAD").Output(); err != nil {
		trackedArgs = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMRTUXB", "-z"}
	}
	tracked, err := exec.Command("git", trackedArgs...).Output()
	if err != nil {
		return nil, "", fmt.Errorf("git diff failed: %w", err)
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return nil, "", fmt.Errorf("git list untracked files failed: %w", err)
	}
	set := make(map[string]struct{})
	for _, raw := range append(bytes.Split(tracked, []byte{0}), bytes.Split(untracked, []byte{0})...) {
		name := filepath.ToSlash(string(raw))
		if name == "" || filepath.Ext(name) != ".go" {
			continue
		}
		absolute := filepath.Join(root, filepath.FromSlash(name))
		info, statErr := os.Stat(absolute)
		if statErr == nil && !info.IsDir() {
			set[name] = struct{}{}
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return nil, "", fmt.Errorf("inspect changed file %s: %w", name, statErr)
		}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, root, nil
}

func selectChangedPaths(changed []string, root string, targets, includes, excludes []string) ([]string, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Git worktree root: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}
	match := func(file, selector string) bool {
		absolute := selector
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(cwd, selector)
		}
		rel, relErr := filepath.Rel(root, absolute)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "." {
			return true
		}
		return file == rel || strings.HasPrefix(file, strings.TrimSuffix(rel, "/")+"/")
	}
	selected := make([]string, 0, len(changed))
	for _, file := range changed {
		inTarget := false
		for _, target := range targets {
			if match(file, target) {
				inTarget = true
				break
			}
		}
		if !inTarget {
			continue
		}
		if len(includes) > 0 {
			included := false
			for _, selector := range includes {
				if match(file, selector) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}
		excluded := false
		for _, selector := range excludes {
			if match(file, selector) {
				excluded = true
				break
			}
		}
		if !excluded {
			selected = append(selected, filepath.Join(root, filepath.FromSlash(file)))
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no changed Go source files found")
	}
	return selected, nil
}
