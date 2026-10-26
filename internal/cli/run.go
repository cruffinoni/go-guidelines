package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/cruffinoni/go-guidelines/internal/fixes"
	"github.com/cruffinoni/go-guidelines/internal/rules"
)

const reportSchemaVersion = "1.0"

type report struct {
	SchemaVersion string          `json:"schema_version"`
	ScannedFiles  int             `json:"scanned_files"`
	Findings      []rules.Finding `json:"findings"`
}

// Run scans Go source files and writes findings in the selected report format.
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
