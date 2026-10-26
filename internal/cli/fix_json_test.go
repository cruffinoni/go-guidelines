package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunJSONPreviewKeepsStdoutValidJSON(t *testing.T) {
	filename := writeFixFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", "--enable-rule", "GBP017", "--fix-dry-run", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() = %d, want 0; stderr=%q", code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; stdout=%q", err, stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Fix preview")) {
		t.Fatalf("stderr = %q, want preview output", stderr.String())
	}
}
