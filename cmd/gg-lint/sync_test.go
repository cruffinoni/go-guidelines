package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncInstructions_UsageErrors(t *testing.T) {
	tests := map[string][]string{
		"unknown harness":              {"sync-instructions", "nope"},
		"missing harness":              {"sync-instructions"},
		"local and user are exclusive": {"sync-instructions", "claude", "--local", "--user"},
		"no user-level file":           {"sync-instructions", "devin"},
		"no project-level file":        {"sync-instructions", "openclaw", "--local"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("HOME", t.TempDir())
			stdout, stderr, code := runCLI(t, newFakeCatalog(t), args...)
			assert.Equal(t, 2, code)
			assert.Empty(t, stdout)
			assert.NotEmpty(t, stderr)
		})
	}
}

func TestSyncInstructions_User(t *testing.T) {
	tests := map[string]struct {
		harness, env, file string
		args               []string
	}{
		"claude by default":  {harness: "claude", env: "CLAUDE_CONFIG_DIR", file: "CLAUDE.md"},
		"claude with --user": {harness: "claude", env: "CLAUDE_CONFIG_DIR", file: "CLAUDE.md", args: []string{"--user"}},
		"codex by default":   {harness: "codex", env: "CODEX_HOME", file: "AGENTS.md"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			userDir, cwd := t.TempDir(), t.TempDir()
			t.Setenv(tc.env, userDir)
			t.Chdir(cwd)

			args := append([]string{"sync-instructions", tc.harness}, tc.args...)
			stdout, stderr, code := runCLI(t, newFakeCatalog(t), args...)
			require.Equalf(t, 0, code, "stderr: %s", stderr)
			assert.Equal(t, "Updated "+tc.harness+" user instructions: "+filepath.Join(userDir, tc.file)+"\n", stdout)
			assert.FileExists(t, filepath.Join(userDir, tc.file))
			assert.NoFileExists(t, filepath.Join(cwd, tc.file))
		})
	}
}

func TestSyncInstructions_UserFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir uses USERPROFILE on Windows
	t.Setenv("CODEX_HOME", "")
	require.NoError(t, os.Mkdir(filepath.Join(home, ".codex"), 0o755))
	t.Chdir(t.TempDir())

	_, stderr, code := runCLI(t, newFakeCatalog(t), "sync-instructions", "codex")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.FileExists(t, filepath.Join(home, ".codex", "AGENTS.md"))
}

func TestSyncInstructions_Local(t *testing.T) {
	userDir, cwd := t.TempDir(), t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", userDir)
	t.Chdir(cwd)

	stdout, stderr, code := runCLI(t, newFakeCatalog(t), "sync-instructions", "claude", "--local")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "Updated claude local instructions: ")
	assert.FileExists(t, filepath.Join(cwd, "CLAUDE.md"))
	assert.NoFileExists(t, filepath.Join(userDir, "CLAUDE.md"))

	stdout, _, code = runCLI(t, newFakeCatalog(t), "sync-instructions", "claude", "--local")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "already current")
}

func TestSyncInstructions_UserDirectoryMustExist(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("CLAUDE_CONFIG_DIR", missing)
	t.Chdir(t.TempDir())

	_, stderr, code := runCLI(t, newFakeCatalog(t), "sync-instructions", "claude")
	assert.Equal(t, 2, code)
	assert.Contains(t, stderr, missing)
	assert.NoDirExists(t, missing)
}

func TestSyncInstructions_UnsupportedScopeSuggestsTheOtherFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	_, stderr, _ := runCLI(t, newFakeCatalog(t), "sync-instructions", "devin")
	assert.Contains(t, stderr, "use --local")
}
