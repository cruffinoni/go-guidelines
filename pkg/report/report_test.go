package report_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/pkg/report"
)

type wireFinding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

type wireDocument struct {
	SchemaVersion string        `json:"schema_version"`
	ScannedFiles  int           `json:"scanned_files"`
	Findings      []wireFinding `json:"findings"`
}

func TestParseFormat(t *testing.T) {
	tests := map[string]struct {
		in      string
		want    report.Format
		wantErr error
	}{
		"text":    {in: "text", want: report.Text},
		"json":    {in: "json", want: report.JSON},
		"unknown": {in: "yaml", wantErr: report.ErrUnknownFormat},
		"empty":   {in: "", wantErr: report.ErrUnknownFormat},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := report.ParseFormat(tc.in)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
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
			{RuleID: "TEST002", Severity: "warning", Message: "bad name", File: "pkg/b.go", Line: 4, Column: 1},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, report.Text.Write(&buf, r))

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, 2)
	for i, finding := range r.Findings {
		assert.Contains(t, lines[i], finding.File)
		assert.Contains(t, lines[i], finding.RuleID)
		assert.Contains(t, lines[i], finding.Severity)
		assert.Contains(t, lines[i], finding.Message)
	}
}

func TestFormatTextWritesNoFindingsLineWhenEmpty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, report.Text.Write(&buf, report.Report{ScannedFiles: 3}))

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, 1)
	assert.NotEmpty(t, lines[0])
	assert.Contains(t, lines[0], "3")
	assert.NotContains(t, lines[0], "[", "empty report must not render a finding")
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

	var decoded wireDocument
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, "1.0", decoded.SchemaVersion)
	assert.Equal(t, 1, decoded.ScannedFiles)
	require.Len(t, decoded.Findings, 1)
	assert.Equal(t, wireFinding{
		RuleID: "TEST001", Severity: "error", Message: "dot import",
		File: "pkg/a.go", Line: 2, Column: 8,
	}, decoded.Findings[0])
	assert.Equal(t, "\n", buf.String()[len(buf.String())-1:], "JSON output must end with a newline")
}

func TestFormatJSONFindingsIsAlwaysAnArray(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, report.JSON.Write(&buf, report.Report{ScannedFiles: 0}))

	var decoded wireDocument
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.NotNil(t, decoded.Findings, "findings must never be null")
	assert.Empty(t, decoded.Findings)
}

func TestFormatUnknown(t *testing.T) {
	var buf bytes.Buffer
	err := report.Format("xml").Write(&buf, report.Report{})
	require.ErrorIs(t, err, report.ErrUnknownFormat)
}

var errDiskFull = errors.New("disk full")

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errDiskFull
}

func TestFormatTextPropagatesWriteError(t *testing.T) {
	tests := map[string]report.Report{
		"findings": {
			ScannedFiles: 1,
			Findings: []report.Finding{
				{RuleID: "TEST001", Severity: "error", Message: "bad", File: "a.go", Line: 1, Column: 1},
			},
		},
		"no findings": {
			ScannedFiles: 2,
		},
	}

	for name, r := range tests {
		t.Run(name, func(t *testing.T) {
			err := report.Text.Write(failWriter{}, r)
			require.ErrorIs(t, err, errDiskFull)
		})
	}
}

func TestFormatJSONPropagatesWriteError(t *testing.T) {
	err := report.JSON.Write(failWriter{}, report.Report{ScannedFiles: 1})
	require.ErrorIs(t, err, errDiskFull)
}
