package report_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/pkg/report"
)

func TestParseFormat(t *testing.T) {
	tests := map[string]struct {
		in      string
		want    report.Format
		wantErr bool
	}{
		"text":    {in: "text", want: report.Text},
		"json":    {in: "json", want: report.JSON},
		"unknown": {in: "yaml", wantErr: true},
		"empty":   {in: "", wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := report.ParseFormat(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatTextWritesOneLinePerFinding(t *testing.T) {
	r := report.Report{
		ScannedFiles: 2,
		Findings: []report.Finding{
			{RuleID: "TEST001", Severity: "error", Message: "dot import", File: "pkg/a.go", Line: 2, Column: 8},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, report.Text.Write(&buf, r))
	assert.Equal(t, "pkg/a.go:2:8: error [TEST001] dot import\n", buf.String())
}

func TestFormatTextWritesNoFindingsLineWhenEmpty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, report.Text.Write(&buf, report.Report{ScannedFiles: 3}))
	assert.Equal(t, "No findings (3 Go files scanned).\n", buf.String())
}

func TestFormatJSONShape(t *testing.T) {
	r := report.Report{
		ScannedFiles: 1,
		Findings: []report.Finding{
			{RuleID: "TEST001", Severity: "error", Message: "dot import", File: "pkg/a.go", Line: 2, Column: 8},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, report.JSON.Write(&buf, r))

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, "1.0", decoded["schema_version"])
	assert.InDelta(t, 1, decoded["scanned_files"], 0)
	findings, ok := decoded["findings"].([]any)
	require.True(t, ok, "findings must decode as an array")
	require.Len(t, findings, 1)
	assert.Equal(t, "\n", buf.String()[len(buf.String())-1:], "JSON output must end with a newline")
}

func TestFormatJSONFindingsIsAlwaysAnArray(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, report.JSON.Write(&buf, report.Report{ScannedFiles: 0}))

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	findings, ok := decoded["findings"].([]any)
	require.True(t, ok, "findings must never be null")
	assert.Empty(t, findings)
}
