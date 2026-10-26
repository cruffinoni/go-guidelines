package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncInstructionsCreatesOnlySelectedProjectFile(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	codex := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(claude, []byte("# My Claude notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte("# My Codex notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	beforeCodex, err := os.ReadFile(codex)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := syncInstructions(root, "claude", os.WriteFile); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("# My Claude notes\n")) || !bytes.Contains(got, []byte(claudeStartMarker)) || !bytes.Contains(got, []byte(claudeEndMarker)) {
		t.Fatalf("CLAUDE.md = %q, expected preserved user content and managed block", got)
	}
	afterCodex, err := os.ReadFile(codex)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterCodex, beforeCodex) {
		t.Fatalf("unselected AGENTS.md changed: before %q, after %q", beforeCodex, afterCodex)
	}
}

func TestSyncInstructionsRefreshesManagedBlockWithoutChangingSurroundingBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	prefix := []byte("user prefix\r\n")
	suffix := []byte("\r\nuser suffix\x00\n")
	initial := append(append(append([]byte{}, prefix...), []byte(claudeStartMarker+"\r\nold\r\n"+claudeEndMarker)...), suffix...)
	if err := os.WriteFile(path, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := syncInstructions(root, "claude", os.WriteFile); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, prefix) || !bytes.HasSuffix(got, suffix) || bytes.Contains(got, []byte("old")) {
		t.Fatalf("surrounding bytes were not preserved or block not refreshed: %q", got)
	}
	first := append([]byte(nil), got...)
	writes := 0
	if _, err := syncInstructions(root, "claude", func(name string, data []byte, perm os.FileMode) error {
		writes++
		return os.WriteFile(name, data, perm)
	}); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatalf("no-op sync performed %d writes", writes)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, again) {
		t.Fatal("repeat sync was not byte-idempotent")
	}
}

func TestSyncInstructionsRefusesSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "global-instructions.md")
	if err := os.WriteFile(outside, []byte("keep these bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Skipf("cannot create symlink in this environment: %v", err)
	}
	if _, err := syncInstructions(root, "claude", os.WriteFile); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("sync error = %v, want explicit symlink refusal", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep these bytes\n" {
		t.Fatalf("external target changed: %q", got)
	}
}

func TestSyncInstructionsCreatesMissingFileAndReportsWriteFailure(t *testing.T) {
	root := t.TempDir()
	if _, err := syncInstructions(root, "codex", os.WriteFile); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "AGENTS.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing instruction file was not created: %v", err)
	}
	failure := errors.New("disk full")
	other := t.TempDir()
	if _, err := syncInstructions(other, "claude", func(string, []byte, os.FileMode) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("write failure = %v, want wrapped disk-full error", err)
	}
	if _, err := os.Stat(filepath.Join(other, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Fatalf("failed write left a target behind: %v", err)
	}
}

func TestSyncInstructionsRejectsMalformedMarkersWithoutWriting(t *testing.T) {
	cases := map[string]string{
		"duplicate start": claudeStartMarker + "\n" + claudeStartMarker + "\n" + claudeEndMarker,
		"unmatched start": claudeStartMarker + "\nuser content",
		"unmatched end":   claudeEndMarker + "\n",
		"reversed":        claudeEndMarker + "\n" + claudeStartMarker,
		"nested":          claudeStartMarker + "\n" + claudeStartMarker + "\n" + claudeEndMarker + "\n" + claudeEndMarker,
		"crossed":         claudeStartMarker + "\n" + codexStartMarker + "\n" + claudeEndMarker + "\n" + codexEndMarker,
		"inline markers":  "prefix " + claudeStartMarker + "\nold\n" + claudeEndMarker + " suffix",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "CLAUDE.md")
			before := []byte("head\n" + content + "\ntail\n")
			if err := os.WriteFile(path, before, 0o600); err != nil {
				t.Fatal(err)
			}
			writes := 0
			_, err := syncInstructions(root, "claude", func(string, []byte, os.FileMode) error { writes++; return nil })
			if err == nil || !strings.Contains(err.Error(), "CLAUDE.md") || !strings.Contains(err.Error(), "marker") {
				t.Fatalf("error = %v, want actionable CLAUDE.md marker error", err)
			}
			if writes != 0 {
				t.Fatalf("malformed file triggered %d writes", writes)
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("malformed file changed: before %q, after %q", before, after)
			}
		})
	}
}
