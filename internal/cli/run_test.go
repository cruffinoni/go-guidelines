package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunJSONReportsGBP001Finding(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\nimport . \"fmt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", filename}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	var result struct {
		SchemaVersion string `json:"schema_version"`
		Findings      []struct {
			RuleID string `json:"rule_id"`
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
		} `json:"findings"`
		ScannedFiles int `json:"scanned_files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v; stdout=%q", err, stdout.String())
	}
	if result.SchemaVersion == "" {
		t.Fatal("schema_version is empty")
	}
	if result.ScannedFiles != 1 || len(result.Findings) != 1 {
		t.Fatalf("result = %#v, want one scanned file and one finding", result)
	}
	finding := result.Findings[0]
	if finding.RuleID != "GBP001" || finding.File != "sample.go" || finding.Line != 2 || finding.Column != 8 {
		t.Fatalf("finding = %#v, want GBP001 at sample.go:2:8", finding)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for a successful JSON report", stderr.String())
	}
}

func TestRunFullDirectoryTextAndJSONReportsHaveFindingParity(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "pkg/bad.go", "package pkg\nimport . \"fmt\"\n")
	writeGoFile(t, root, "pkg/clean.go", "package pkg\n")
	var textOut, textErr, jsonOut, jsonErr bytes.Buffer
	textCode := Run([]string{root}, &textOut, &textErr)
	jsonCode := Run([]string{"--format=json", root}, &jsonOut, &jsonErr)
	if textCode != 1 || jsonCode != 1 {
		t.Fatalf("text/json exit codes = %d/%d, want both 1; stderr=%q/%q", textCode, jsonCode, textErr.String(), jsonErr.String())
	}
	var decoded struct {
		ScannedFiles int `json:"scanned_files"`
		Findings     []struct {
			RuleID string `json:"rule_id"`
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(jsonOut.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON report: %v; output=%q", err, jsonOut.String())
	}
	if decoded.ScannedFiles != 2 || len(decoded.Findings) != 1 {
		t.Fatalf("JSON counts = %d files, %d findings; want 2 and 1", decoded.ScannedFiles, len(decoded.Findings))
	}
	finding := decoded.Findings[0]
	for _, want := range []string{finding.RuleID, finding.File + ":2:8", "avoid dot imports"} {
		if !strings.Contains(textOut.String(), want) {
			t.Errorf("text report %q lacks JSON finding detail %q", textOut.String(), want)
		}
	}
	if finding.RuleID != "GBP001" || finding.Line != 2 || finding.Column != 8 {
		t.Fatalf("JSON finding = %#v, want GBP001 at line 2 column 8", finding)
	}
}

func TestRunDisabledGBP001HasNoFindings(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\nimport . \"fmt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", "--disable-rule", "GBP001", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	var result struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v; stdout=%q", err, stdout.String())
	}
	if len(result.Findings) != 0 {
		t.Fatalf("findings = %d, want 0", len(result.Findings))
	}
}

func TestRunTextReportIsDeterministicAndNamesFinding(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\nimport . \"fmt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var first, second bytes.Buffer
	if code := Run([]string{filename}, &first, &bytes.Buffer{}); code != 1 {
		t.Fatalf("first Run() exit code = %d, want 1", code)
	}
	if code := Run([]string{filename}, &second, &bytes.Buffer{}); code != 1 {
		t.Fatalf("second Run() exit code = %d, want 1", code)
	}
	if first.String() != second.String() {
		t.Fatalf("text report is not deterministic:\nfirst: %q\nsecond: %q", first.String(), second.String())
	}
	for _, expected := range []string{"GBP001", "sample.go:2:8", "avoid dot imports"} {
		if !strings.Contains(first.String(), expected) {
			t.Errorf("text report %q does not contain %q", first.String(), expected)
		}
	}
}

func TestRunOperationalErrorUsesExitCodeTwoWithoutJSONDiagnostics(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", filepath.Join(t.TempDir(), "missing.go")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run() exit code = %d, want 2; stderr=%q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty so JSON output is not contaminated", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("stderr is empty, want operational diagnostic")
	}
}
