package git_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/pkg/git"
)

// runGit runs a Git subcommand in root, failing the test on error.
func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, output)
}

// initRepo creates a Git repository with one committed file and returns its
// root. User and email are set locally so the test does not depend on
// global Git configuration.
func initRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.invalid")
	runGit(t, root, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("base\n"), 0o600))
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-qm", "initial")
	return root
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o700))
	require.NoError(t, os.WriteFile(name, []byte(content), 0o600))
}

func TestRootReturnsWorktreeTopLevel(t *testing.T) {
	root := initRepo(t)
	nested := filepath.Join(root, "pkg")
	require.NoError(t, os.Mkdir(nested, 0o700))

	got, err := git.Root(t.Context(), nested)
	require.NoError(t, err)

	wantRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	gotRoot, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	assert.Equal(t, wantRoot, gotRoot)
}

func TestRootErrorsOutsideAWorktree(t *testing.T) {
	_, err := git.Root(t.Context(), t.TempDir())
	assert.True(t, errors.Is(err, git.ErrNotRepository))
}

func TestChangedFilesIncludesStagedFiles(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "staged.go", "package sample\n")
	runGit(t, root, "add", "staged.go")

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "staged.go")}, got)
}

func TestChangedFilesIncludesUnstagedModifications(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "tracked.go", "package sample\n")
	runGit(t, root, "add", "tracked.go")
	runGit(t, root, "commit", "-qm", "add tracked source")
	writeFile(t, root, "tracked.go", "package sample\n// unstaged change\n")

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "tracked.go")}, got)
}

func TestChangedFilesIncludesStagedAndUnstagedTogether(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "tracked.go", "package sample\n")
	runGit(t, root, "add", "tracked.go")
	runGit(t, root, "commit", "-qm", "add tracked source")
	writeFile(t, root, "tracked.go", "package sample\n// staged\n")
	runGit(t, root, "add", "tracked.go")
	writeFile(t, root, "tracked.go", "package sample\n// staged\n// unstaged\n")

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "tracked.go")}, got)
}

func TestChangedFilesIncludesUntrackedFiles(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "new.go", "package sample\n")

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "new.go")}, got)
}

func TestChangedFilesOmitsDeletedAndIgnoredFiles(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "gone.go", "package sample\n")
	runGit(t, root, "add", "gone.go")
	runGit(t, root, "commit", "-qm", "add source")
	require.NoError(t, os.Remove(filepath.Join(root, "gone.go")))
	writeFile(t, root, "new.go", "package sample\n")
	writeFile(t, root, "ignored.go", "package sample\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored.go\n"), 0o600))
	runGit(t, root, "add", ".gitignore")

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, ".gitignore"), filepath.Join(root, "new.go")}, got)
}

func TestChangedFilesUsesRenameDestination(t *testing.T) {
	root := initRepo(t)
	writeFile(t, root, "old.go", "package sample\n")
	runGit(t, root, "add", "old.go")
	runGit(t, root, "commit", "-qm", "add source")
	require.NoError(t, os.Rename(filepath.Join(root, "old.go"), filepath.Join(root, "renamed.go")))

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "renamed.go")}, got)
}

func TestChangedFilesHandlesUnbornRepository(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	writeFile(t, root, "first.go", "package sample\n")
	runGit(t, root, "add", "first.go")

	got, err := git.ChangedFiles(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "first.go")}, got)
}
