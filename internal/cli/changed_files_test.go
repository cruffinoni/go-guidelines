package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunChangedFilesScansStagedGoFilesRelativeToRepository(t *testing.T) {
	root := initGitRepo(t)
	writeGoFile(t, root, "pkg/staged.go", "package pkg\nimport . \"fmt\"\n")
	git(t, root, "add", "pkg/staged.go")
	workdir := filepath.Join(root, "pkg")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(workdir); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--format=json", "--changed-files", "."}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() exit code = %d, want finding exit 1; stderr=%q", code, stderr.String())
	}
	var result struct {
		ScannedFiles int `json:"scanned_files"`
		Findings     []struct {
			File string `json:"file"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v; stdout=%q", err, stdout.String())
	}
	if result.ScannedFiles != 1 || len(result.Findings) != 1 || result.Findings[0].File != "pkg/staged.go" {
		t.Fatalf("result = %#v, want one finding in repo-relative pkg/staged.go", result)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "test@example.invalid")
	git(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "README.md")
	git(t, root, "commit", "-qm", "initial")
	return root
}

func writeGoFile(t *testing.T, root, relative, content string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestGitChangedFilesHandlesDeletedIgnoredAndUntracked(t *testing.T) {
	root := initGitRepo(t)
	writeGoFile(t, root, "gone.go", "package sample\n")
	git(t, root, "add", "gone.go")
	git(t, root, "commit", "-qm", "add source")
	if err := os.Remove(filepath.Join(root, "gone.go")); err != nil {
		t.Fatal(err)
	}
	writeGoFile(t, root, "new.go", "package sample\n")
	writeGoFile(t, root, "ignored.go", "package sample\n")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", ".gitignore")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	files, _, err := gitChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "new.go" {
		t.Fatalf("changed files = %v, want [new.go]", files)
	}
}

func TestGitChangedFilesErrorsOutsideGitWorktree(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := gitChangedFiles(); err == nil {
		t.Fatal("gitChangedFiles() succeeded outside a Git worktree")
	}
}

func TestGitChangedFilesIncludesTrackedUnstagedModification(t *testing.T) {
	root := initGitRepo(t)
	writeGoFile(t, root, "tracked.go", "package sample\n")
	git(t, root, "add", "tracked.go")
	git(t, root, "commit", "-qm", "add tracked source")
	writeGoFile(t, root, "tracked.go", "package sample\n// unstaged change\n")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	files, _, err := gitChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "tracked.go" {
		t.Fatalf("changed files = %v, want [tracked.go]", files)
	}
}

func TestGitChangedFilesIncludesStagedAndUnstagedChangesTogether(t *testing.T) {
	root := initGitRepo(t)
	writeGoFile(t, root, "tracked.go", "package sample\n")
	git(t, root, "add", "tracked.go")
	git(t, root, "commit", "-qm", "add tracked source")
	writeGoFile(t, root, "tracked.go", "package sample\n// staged\n")
	git(t, root, "add", "tracked.go")
	writeGoFile(t, root, "tracked.go", "package sample\n// staged\n// unstaged\n")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	files, _, err := gitChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "tracked.go" {
		t.Fatalf("changed files = %v, want [tracked.go]", files)
	}
}

func TestGitChangedFilesUsesRenameDestination(t *testing.T) {
	root := initGitRepo(t)
	writeGoFile(t, root, "old.go", "package sample\n")
	git(t, root, "add", "old.go")
	git(t, root, "commit", "-qm", "add source")
	if err := os.Rename(filepath.Join(root, "old.go"), filepath.Join(root, "renamed.go")); err != nil {
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
	files, _, err := gitChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "renamed.go" {
		t.Fatalf("changed files = %v, want rename destination [renamed.go]", files)
	}
}

func TestGitChangedFilesHandlesUnbornRepository(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q")
	writeGoFile(t, root, "first.go", "package sample\n")
	git(t, root, "add", "first.go")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	files, _, err := gitChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "first.go" {
		t.Fatalf("changed files = %v, want [first.go]", files)
	}
}

func TestSelectChangedPathsAppliesTargetIncludeAndExclude(t *testing.T) {
	root := initGitRepo(t)
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	got, err := selectChangedPaths([]string{"pkg/a.go", "pkg/skip.go", "other/b.go"}, root, []string{"."}, []string{"pkg"}, []string{"pkg/skip.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(root, "pkg/a.go") {
		t.Fatalf("selected = %v, want only pkg/a.go", got)
	}
}

func TestSelectChangedPathsHandlesSymlinkedRepositoryRoot(t *testing.T) {
	actualRoot := t.TempDir()
	linkParent := t.TempDir()
	linkedRoot := filepath.Join(linkParent, "repo")
	if err := os.Symlink(actualRoot, linkedRoot); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(actualRoot); err != nil {
		t.Fatal(err)
	}

	got, err := selectChangedPaths([]string{"pkg/a.go"}, linkedRoot, []string{"."}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(actualRoot, "pkg", "a.go") {
		t.Fatalf("selected = %v, want path beneath resolved root %q", got, actualRoot)
	}
}
