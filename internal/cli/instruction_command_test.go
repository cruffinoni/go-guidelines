package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInstructionSyncUpdatesSelectedProviderAtGitRoot(t *testing.T) {
	root := initGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("Claude user text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("Codex user text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := RunInstructionSync([]string{"--provider", "codex"}, &stdout, &stderr); code != 0 {
		t.Fatalf("RunInstructionSync() = %d, stderr=%q", code, stderr.String())
	}
	claude, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(claude, []byte("gg-lint:")) || !bytes.Contains(agents, []byte(codexStartMarker)) {
		t.Fatalf("provider selection did not isolate update: CLAUDE=%q AGENTS=%q", claude, agents)
	}
	if stderr.Len() != 0 || !strings.Contains(stdout.String(), "AGENTS.md") {
		t.Fatalf("unexpected command output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunScanNeverCreatesInstructionFiles(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "main.go", "package main\n")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"."}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() = %d, stderr=%q", code, stderr.String())
	}
	for _, filename := range []string{"CLAUDE.md", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(root, filename)); !os.IsNotExist(err) {
			t.Errorf("scan created %s: stat error = %v", filename, err)
		}
	}
}

func TestRunInstructionSyncRequiresProvider(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RunInstructionSync(nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "--provider") {
		t.Fatalf("RunInstructionSync() = %d, stderr=%q; want required-provider error", code, stderr.String())
	}
}
