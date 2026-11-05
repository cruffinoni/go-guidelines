package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
)

// jsonReport mirrors pkg/report's JSON document, for decoding in tests.
type jsonReport struct {
	SchemaVersion string `json:"schema_version"`
	ScannedFiles  int    `json:"scanned_files"`
	Findings      []struct {
		RuleID   string `json:"rule_id"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
		File     string `json:"file"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
	} `json:"findings"`
}

func TestScan_ExitCodes(t *testing.T) {
	cases := map[string]struct {
		source string
		args   []string
		want   int
	}{
		"no findings": {
			source: "package x\n",
			args:   nil,
			want:   0,
		},
		"error finding exits 1": {
			source: "package x\n\nvar Bad = 1\n",
			args:   nil,
			want:   1,
		},
		"fail-on none disables the threshold": {
			source: "package x\n\nvar Bad = 1\n",
			args:   []string{"--fail-on=none"},
			want:   0,
		},
		"info finding below the default threshold": {
			source: "package x\n\nvar Watch = 1\n",
			args:   nil,
			want:   0,
		},
		"fail-on info catches the info finding": {
			source: "package x\n\nvar Watch = 1\n",
			args:   []string{"--fail-on=info"},
			want:   1,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			writeModule(t, map[string]string{"a.go": tc.source})
			cat := newFakeCatalog(t)
			args := append([]string{"scan"}, tc.args...)
			_, _, code := runCLI(t, cat, args...)
			assert.Equal(t, tc.want, code)
		})
	}
}

func TestScan_DisabledRuleProducesNoFinding(t *testing.T) {
	writeModule(t, map[string]string{"a.go": "package x\n\nvar Bad = 1\n"})
	cat := newFakeCatalog(t)
	stdout, _, code := runCLI(t, cat, "scan", "--format=json", "--fail-on=none", "--disable-rule=TEST001")
	require.Equal(t, 0, code)

	var doc jsonReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Empty(t, doc.Findings)
}

func TestScan_JSONSmoke(t *testing.T) {
	writeModule(t, map[string]string{"a.go": "package x\n\nvar Bad = 1\n"})
	cat := newFakeCatalog(t)
	stdout, _, code := runCLI(t, cat, "scan", "--format=json", "--fail-on=none", ".")
	require.Equal(t, 0, code)

	var doc jsonReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	require.Len(t, doc.Findings, 1)
	assert.Equal(t, "TEST001", doc.Findings[0].RuleID)
	assert.Equal(t, "a.go", doc.Findings[0].File)
}

func TestScan_ConfigFlagPrecedence(t *testing.T) {
	dir := writeModule(t, map[string]string{"a.go": "package x\n\nvar Bad = 1\n"})
	cat := newFakeCatalog(t)

	configPath := filepath.Join(dir, "gg-lint.yaml")
	config := "format: json\nfail_on: none\nrules:\n  TEST001: false\n"
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

	// Config disables TEST001 and allows the finding through (fail_on = none).
	stdout, _, code := runCLI(t, cat, "scan", "--config="+configPath)
	require.Equal(t, 0, code)
	var doc jsonReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	assert.Empty(t, doc.Findings)

	// --enable-rule overrides the config file's disable.
	stdout, _, code = runCLI(t, cat, "scan", "--config="+configPath, "--enable-rule=TEST001")
	require.Equal(t, 0, code, "config fail_on=none still wins without a --fail-on flag")
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	require.Len(t, doc.Findings, 1)
	assert.Equal(t, "TEST001", doc.Findings[0].RuleID)

	// --fail-on overrides the config file's fail_on.
	_, _, code = runCLI(t, cat, "scan", "--config="+configPath, "--enable-rule=TEST001", "--fail-on=error")
	assert.Equal(t, 1, code)

	// --format overrides the config file's format.
	stdout, _, code = runCLI(t, cat, "scan", "--config="+configPath, "--enable-rule=TEST001", "--format=text")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "TEST001")
	assert.False(t, strings.HasPrefix(strings.TrimSpace(stdout), "{"), "want text output, got %q", stdout)
}

func TestScan_UserConfigOutsideGit(t *testing.T) {
	writeModule(t, map[string]string{"a.go": "package x\n\nvar Bad = 1\n"})
	cat := newFakeCatalog(t)

	userPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "gg-lint", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(userPath), 0o755))
	require.NoError(t, os.WriteFile(userPath, []byte("format: json\nfail_on: none\n"), 0o644))

	stdout, _, code := runCLI(t, cat, "scan", "--enable-rule=TEST001")
	require.Equal(t, 0, code, "user config fail_on=none must apply")
	var doc jsonReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	require.Len(t, doc.Findings, 1)
}

// initGitModule creates a Git repository holding a Go module with a.go
// committed, and returns its root.
func initGitModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o644))
	}
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, output)
	}
	write("go.mod", "module example.test/repo\n\ngo 1.27\n")
	write("a.go", "package x\n")
	git("init", "-q")
	git("config", "user.email", "test@example.invalid")
	git("config", "user.name", "Test")
	git("add", ".")
	git("commit", "-qm", "initial")
	return root
}

// TestScan_GitChangesUsesTargetRepository verifies the Git worktree comes
// from the scanned path, not the working directory.
func TestScan_GitChangesUsesTargetRepository(t *testing.T) {
	writeModule(t, map[string]string{"a.go": "package x\n"})
	repo := initGitModule(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "b.go"), []byte("package x\n"), 0o644))
	cat := newFakeCatalog(t)

	for _, target := range []string{repo, repo + "/..."} {
		stdout, stderr, code := runCLI(t, cat, "scan", "--git-changes", "--format=json", target)
		require.Equalf(t, 0, code, "target %s: %s", target, stderr)
		var doc jsonReport
		require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
		assert.Equal(t, 1, doc.ScannedFiles, "target %s", target)
	}
}

func TestScan_OperationalErrors(t *testing.T) {
	cases := map[string]struct {
		source string
		args   []string
	}{
		"unknown rule in --enable-rule": {
			source: "package x\n",
			args:   []string{"--enable-rule=NOPE"},
		},
		"unknown rule in --disable-rule": {
			source: "package x\n",
			args:   []string{"--disable-rule=NOPE"},
		},
		"same rule enabled and disabled": {
			source: "package x\n",
			args:   []string{"--enable-rule=TEST001", "--disable-rule=TEST001"},
		},
		"unsupported format": {
			source: "package x\n",
			args:   []string{"--format=xml"},
		},
		"unknown flag --fix": {
			source: "package x\n",
			args:   []string{"--fix"},
		},
		"missing --config file": {
			source: "package x\n",
			args:   []string{"--config=missing.yaml"},
		},
		"config and no-config together": {
			source: "package x\n",
			args:   []string{"--config=missing.yaml", "--no-config"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			writeModule(t, map[string]string{"a.go": tc.source})
			cat := newFakeCatalog(t)
			args := append([]string{"scan"}, tc.args...)
			stdout, stderr, code := runCLI(t, cat, args...)
			assert.Equal(t, 2, code)
			assert.Empty(t, stdout, "stdout must stay empty on an operational error")
			assert.NotEmpty(t, stderr)
		})
	}
}

func TestScan_UnknownRuleInConfig(t *testing.T) {
	dir := writeModule(t, map[string]string{"a.go": "package x\n"})
	cat := newFakeCatalog(t)

	configPath := filepath.Join(dir, "gg-lint.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("rules:\n  NOPE: false\n"), 0o644))

	stdout, stderr, code := runCLI(t, cat, "scan", "--config="+configPath)
	assert.Equal(t, 2, code)
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}

func writeRepoFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, output)
}

func decodeJSONReport(t *testing.T, stdout string) jsonReport {
	t.Helper()
	var doc jsonReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	return doc
}

func hasRule(doc jsonReport, id string) bool {
	for _, finding := range doc.Findings {
		if finding.RuleID == id {
			return true
		}
	}
	return false
}

func TestScan_ConfigResolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	tests := map[string]struct {
		setup     func(t *testing.T) []string
		wantCode  int
		checkRule bool
		wantRule  bool
	}{
		"caller config does not leak": {
			setup: func(t *testing.T) []string {
				repoA := initGitModule(t)
				repoB := initGitModule(t)
				writeRepoFile(t, repoA, "gg-lint.yaml", "fail_on: none\nrules:\n  TEST001: false\n")
				writeRepoFile(t, repoB, "a.go", "package x\n\nvar Bad = 1\n")
				t.Chdir(repoA)
				return []string{"scan", "--fail-on=none", "--format=json", repoB}
			},
			wantCode:  0,
			checkRule: true,
			wantRule:  true,
		},
		"target config applies": {
			setup: func(t *testing.T) []string {
				repoA := initGitModule(t)
				repoB := initGitModule(t)
				writeRepoFile(t, repoB, "gg-lint.yaml", "fail_on: none\nrules:\n  TEST001: false\n")
				writeRepoFile(t, repoB, "a.go", "package x\n\nvar Bad = 1\n")
				t.Chdir(repoA)
				return []string{"scan", "--fail-on=none", "--format=json", repoB}
			},
			wantCode:  0,
			checkRule: true,
			wantRule:  false,
		},
		"no-config ignores repository config": {
			setup: func(t *testing.T) []string {
				repo := initGitModule(t)
				writeRepoFile(t, repo, "gg-lint.yaml", "fail_on: none\nrules:\n  TEST001: false\n")
				writeRepoFile(t, repo, "a.go", "package x\n\nvar Bad = 1\n")
				t.Chdir(repo)
				return []string{"scan", "--no-config", "--fail-on=none", "--format=json", "."}
			},
			wantCode:  0,
			checkRule: true,
			wantRule:  true,
		},
		"no-config ignores user config": {
			setup: func(t *testing.T) []string {
				writeModule(t, map[string]string{"a.go": "package x\n\nvar Bad = 1\n"})
				userPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "gg-lint", "config.yaml")
				require.NoError(t, os.MkdirAll(filepath.Dir(userPath), 0o755))
				require.NoError(t, os.WriteFile(userPath, []byte("fail_on: none\n"), 0o644))
				return []string{"scan", "--no-config", "--format=json", "."}
			},
			wantCode:  1,
			checkRule: true,
			wantRule:  true,
		},
		"multiple worktrees require explicit config": {
			setup: func(t *testing.T) []string {
				repoA := initGitModule(t)
				repoB := initGitModule(t)
				t.Chdir(repoA)
				return []string{"scan", repoA, repoB}
			},
			wantCode: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := tc.setup(t)
			stdout, stderr, code := runCLI(t, newFakeCatalog(t), args...)
			assert.Equal(t, tc.wantCode, code, "stderr: %s", stderr)
			if tc.checkRule {
				doc := decodeJSONReport(t, stdout)
				assert.Equal(t, tc.wantRule, hasRule(doc, "TEST001"))
			}
		})
	}
}

func TestResolveConfigPathRejectsMultipleWorktrees(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repoA := initGitModule(t)
	repoB := initGitModule(t)
	t.Chdir(repoA)

	_, err := resolveConfigPath(t.Context(), []string{repoA, repoB}, "", false)
	require.ErrorIs(t, err, errMultipleWorktrees)
}

func newGitScopeRepo(t *testing.T) string {
	t.Helper()
	root := initGitModule(t)
	writeRepoFile(t, root, "testdata/bad.go", "package x\n\nfunc broken( {\n")
	writeRepoFile(t, root, "excluded.go", "//go:build ignore\n\npackage x\n\nimport . \"fmt\"\n\nvar _ = Sprintf\n")
	writeRepoFile(t, root, "gen.go", "// Code generated by test. DO NOT EDIT.\n\npackage x\n")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-qm", "fixtures")
	return root
}

func TestScan_GitChangesMatchesDirectoryScanScope(t *testing.T) {
	tests := map[string]struct {
		mutate      func(t *testing.T, root string)
		wantScanned int
		wantRule    bool
	}{
		"clean worktree is a no-op": {
			wantScanned: 0,
		},
		"testdata-only change": {
			mutate: func(t *testing.T, root string) {
				writeRepoFile(t, root, "testdata/bad.go", "package x\n\nfunc broken( {\n// changed\n")
			},
			wantScanned: 0,
		},
		"build-tag-only change": {
			mutate: func(t *testing.T, root string) {
				writeRepoFile(t, root, "excluded.go", "//go:build ignore\n\npackage x\n\nimport . \"fmt\"\n\nvar _ = Sprintf\n// changed\n")
			},
			wantScanned: 0,
		},
		"generated-only change": {
			mutate: func(t *testing.T, root string) {
				writeRepoFile(t, root, "gen.go", "// Code generated by test. DO NOT EDIT.\n\npackage x\n// changed\n")
			},
			wantScanned: 0,
		},
		"normal change still reports": {
			mutate: func(t *testing.T, root string) {
				writeRepoFile(t, root, "a.go", "package x\n\nvar Bad = 1\n")
				writeRepoFile(t, root, "testdata/bad.go", "package x\n\nfunc broken( {\n// changed again\n")
			},
			wantScanned: 1,
			wantRule:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			root := newGitScopeRepo(t)
			t.Chdir(root)
			if tc.mutate != nil {
				tc.mutate(t, root)
			}

			stdout, stderr, code := runCLI(t, newFakeCatalog(t),
				"scan", "--git-changes", "--fail-on=none", "--format=json", ".")
			require.Equalf(t, 0, code, "stderr: %s", stderr)
			doc := decodeJSONReport(t, stdout)
			assert.Equal(t, tc.wantScanned, doc.ScannedFiles)
			assert.Equal(t, tc.wantRule, hasRule(doc, "TEST001"))
		})
	}
}

func TestScan_PackageDocFindingUsesInScopeFile(t *testing.T) {
	writeModule(t, map[string]string{
		"a.go": "package x\n\nvar A = 1\n",
		"b.go": "package x\n\nvar B = 1\n",
	})
	cat, err := catalog.Builtin()
	require.NoError(t, err)

	tests := map[string]struct {
		args []string
	}{
		"single file": {
			args: []string{"b.go"},
		},
		"excluded representative": {
			args: []string{"--exclude=a.go", "."},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"scan", "--fail-on=none", "--format=json"}, tc.args...)
			stdout, stderr, code := runCLI(t, cat, args...)
			require.Equalf(t, 0, code, "stderr: %s", stderr)
			doc := decodeJSONReport(t, stdout)
			require.Len(t, doc.Findings, 1)
			assert.Equal(t, "GBP002", doc.Findings[0].RuleID)
			assert.Equal(t, "b.go", doc.Findings[0].File)
		})
	}
}

func TestScan_SingleNonTestFileDoesNotLoadBrokenTests(t *testing.T) {
	writeModule(t, map[string]string{
		"ok.go":          "package x\n",
		"broken_test.go": "package x\n\nfunc broken( {\n",
	})
	_, stderr, code := runCLI(t, newFakeCatalog(t), "scan", "--fail-on=none", "ok.go")
	assert.Equal(t, 0, code, "stderr: %s", stderr)
}
