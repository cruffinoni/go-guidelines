package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	configPath := filepath.Join(dir, "gg-lint.toml")
	config := "format = \"json\"\nfail_on = \"none\"\n\n[rules]\nTEST001 = false\n"
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
	assert.True(t, strings.Contains(stdout, "[TEST001]"), "want text output, got %q", stdout)
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

	configPath := filepath.Join(dir, "gg-lint.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[rules]\nNOPE = false\n"), 0o644))

	stdout, stderr, code := runCLI(t, cat, "scan", "--config="+configPath)
	assert.Equal(t, 2, code)
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}
