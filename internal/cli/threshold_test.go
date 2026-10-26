package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunFailOnNoneNeverFailsForFindings(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\nimport . \"fmt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", "--fail-on", "none", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d, want 0 when failure threshold is disabled; stderr=%q", code, stderr.String())
	}
}
