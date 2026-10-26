package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunEnablesGCM009ThroughCLIWithInfoSeverity(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "sample.go")
	source := "package sample\n// GetName returns the name.\nfunc GetName() string { return \"x\" }\n"
	if err := os.WriteFile(filename, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", "--fail-on=error", "--enable-rule", "GCM009", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 for info-only finding; stderr=%q", code, stderr.String())
	}
	var result struct {
		Findings []struct {
			RuleID   string `json:"rule_id"`
			Severity string `json:"severity"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v; stdout=%q", err, stdout.String())
	}
	if len(result.Findings) != 1 || result.Findings[0].RuleID != "GCM009" || result.Findings[0].Severity != "info" {
		t.Fatalf("findings = %#v, want one info GCM009", result.Findings)
	}
}

func TestRunRejectsRemovedRules(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"GBP007", "GBP020"} {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"--enable-rule", id, filename}, &stdout, &stderr); code != 2 {
			t.Errorf("Run(--enable-rule %s) = %d, want 2; stderr=%q", id, code, stderr.String())
		}
	}
}
