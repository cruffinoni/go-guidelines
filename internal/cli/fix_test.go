package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoesNotChangeFixableSourceWithoutExplicitOptIn(t *testing.T) {
	filename := writeFixFixture(t)
	before, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--enable-rule", "GBP017", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 for warning-level finding; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "GBP017") {
		t.Fatalf("scan output = %q, want GBP017 finding", stdout.String())
	}
	after, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("source changed without explicit fix opt-in:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestRunPreviewShowsFixWithoutChangingSource(t *testing.T) {
	filename := writeFixFixture(t)
	before, _ := os.ReadFile(filename)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--enable-rule", "GBP017", "--fix-dry-run", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 for warning-level preview; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "var (\n") || !strings.Contains(stdout.String(), "preview") {
		t.Fatalf("preview output = %q, want grouped declaration and preview notice", stdout.String())
	}
	after, _ := os.ReadFile(filename)
	if !bytes.Equal(after, before) {
		t.Fatal("preview modified source")
	}
}

func TestRunAppliesExplicitlyRequestedFixAtomically(t *testing.T) {
	filename := writeFixFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--enable-rule", "GBP017", "--fix", filename}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 after applying fix; stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "Applied 1 fix") {
		t.Fatalf("apply output = %q, want visible applied-fix notice", stdout.String())
	}
	after, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "var (\n	alpha int\n	beta string\n	gamma bool\n)") {
		t.Fatalf("fixed source = %q, want grouped declarations", after)
	}
}

func TestRunRejectsConflictingFixModes(t *testing.T) {
	filename := writeFixFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--fix", "--fix-dry-run", filename}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "cannot be used together") {
		t.Fatalf("Run() = %d, stderr=%q; want conflicting-mode usage error", code, stderr.String())
	}
}

func writeFixFixture(t *testing.T) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "sample.go")
	source := "package sample\nvar alpha int\nvar beta string\nvar gamma bool\n"
	if err := os.WriteFile(filename, []byte(source), 0o640); err != nil {
		t.Fatal(err)
	}
	return filename
}
