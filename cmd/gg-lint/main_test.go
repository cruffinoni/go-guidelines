package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionPrintsBuildMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execute([]string{"version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("execute(version) = %d, want 0; stderr: %s", code, stderr.String())
	}
	for _, want := range []string{"gg-lint dev", "commit: unknown", "built: unknown"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("version output %q does not contain %q", stdout.String(), want)
		}
	}
	if stderr.Len() != 0 {
		t.Errorf("version wrote unexpected stderr: %s", stderr.String())
	}
}
