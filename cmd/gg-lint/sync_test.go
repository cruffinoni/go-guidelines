package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHarness_UsageErrors(t *testing.T) {
	tests := map[string][]string{
		"unknown harness":              {"nope"},
		"missing harness":              nil,
		"extra harness":                {"claude", "codex"},
		"local and user are exclusive": {"claude", "--local", "--user"},
		"no project-level file":        {"openclaw", "--local"},
	}
	for _, action := range []string{"setup", "remove"} {
		for name, args := range tests {
			t.Run(action+"/"+name, func(t *testing.T) {
				t.Chdir(t.TempDir())
				t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
				stdout, stderr, code := runCLI(t, newFakeCatalog(t), append([]string{"harness", action}, args...)...)
				assert.Equal(t, 2, code)
				assert.Empty(t, stdout)
				assert.NotEmpty(t, stderr)
			})
		}
	}
}

func TestHarnessSetup_User(t *testing.T) {
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

			args := append([]string{"harness", "setup", tc.harness}, tc.args...)
			stdout, stderr, code := runCLI(t, newFakeCatalog(t), args...)
			require.Equalf(t, 0, code, "stderr: %s", stderr)
			assert.Equal(t, "Updated "+tc.harness+" user instructions: "+filepath.Join(userDir, tc.file)+"\n", stdout)
			assert.FileExists(t, filepath.Join(userDir, tc.file))
			assert.NoFileExists(t, filepath.Join(cwd, tc.file))
		})
	}
}

func TestHarnessSetup_UserFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir uses USERPROFILE on Windows
	t.Setenv("CODEX_HOME", "")
	require.NoError(t, os.Mkdir(filepath.Join(home, ".codex"), 0o755))
	t.Chdir(t.TempDir())

	_, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "codex")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.FileExists(t, filepath.Join(home, ".codex", "AGENTS.md"))
}

func TestHarnessSetup_UserDoesNotRequireGit(t *testing.T) {
	userDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", userDir)
	t.Setenv("PATH", t.TempDir()) // no git anywhere
	t.Chdir(t.TempDir())

	_, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "claude")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.FileExists(t, filepath.Join(userDir, "CLAUDE.md"))
}

func TestHarnessSetup_LocalReportsMissingGit(t *testing.T) {
	root := initGitModule(t) // git runs before PATH is cleared
	t.Setenv("PATH", t.TempDir())
	t.Chdir(root)

	_, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "claude", "--local")
	assert.Equal(t, 2, code)
	assert.Contains(t, stderr, "git")
	assert.NoFileExists(t, filepath.Join(root, "CLAUDE.md"))
}

func TestHarnessSetup_Local(t *testing.T) {
	userDir, cwd := t.TempDir(), t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", userDir)
	t.Chdir(cwd)

	stdout, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "claude", "--local")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "Updated claude local instructions: ")
	assert.FileExists(t, filepath.Join(cwd, "CLAUDE.md"))
	assert.NoFileExists(t, filepath.Join(userDir, "CLAUDE.md"))

	stdout, _, code = runCLI(t, newFakeCatalog(t), "harness", "setup", "claude", "--local")
	require.Equal(t, 0, code)
	assert.Contains(t, stdout, "already current")
}

func TestHarnessSetup_LocalUsesTheRepositoryRoot(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	root := initGitModule(t)
	sub := filepath.Join(root, "pkg", "foo")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	t.Chdir(sub)

	_, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "claude", "--local")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.FileExists(t, filepath.Join(root, "CLAUDE.md"))
	assert.NoFileExists(t, filepath.Join(sub, "CLAUDE.md"))
}

func TestHarnessSetup_UserDirectoryMustExist(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("CLAUDE_CONFIG_DIR", missing)
	t.Chdir(t.TempDir())

	_, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "claude")
	assert.Equal(t, 2, code)
	assert.Contains(t, stderr, missing)
	assert.Contains(t, stderr, "create that directory first")
	assert.Contains(t, stderr, "use --local")
	assert.NoDirExists(t, missing)
}

func TestHarnessSetup_UnsupportedScopeSuggestsTheOtherFlag(t *testing.T) {
	t.Chdir(t.TempDir())
	_, stderr, _ := runCLI(t, newFakeCatalog(t), "harness", "setup", "openclaw", "--local")
	assert.Contains(t, stderr, "use --user")
}

func TestHarnessSetup_DevinUser(t *testing.T) {
	userDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(userDir, "devin"), 0o755))
	t.Setenv("XDG_CONFIG_HOME", userDir)
	t.Setenv("AppData", userDir) // os.UserConfigDir uses AppData on Windows
	t.Chdir(t.TempDir())

	_, stderr, code := runCLI(t, newFakeCatalog(t), "harness", "setup", "devin")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.FileExists(t, filepath.Join(userDir, "devin", "AGENTS.md"))
}

func TestHarnessHelp(t *testing.T) {
	cat := newFakeCatalog(t)
	stdout, stderr, code := runCLI(t, cat, "--help")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "harness")
	assert.NotContains(t, stdout, "sync-instructions")

	stdout, stderr, code = runCLI(t, cat, "harness", "--help")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "setup")
	assert.Contains(t, stdout, "remove")

	stdout, stderr, code = runCLI(t, cat, "harness")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "Available Commands:")

	stdout, stderr, code = runCLI(t, cat, "harness", "delete")
	assert.Equal(t, 2, code)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "unknown command")

	for _, action := range []string{"setup", "remove"} {
		stdout, stderr, code = runCLI(t, cat, "harness", action, "--help")
		require.Equalf(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, "gg-lint harness "+action+" <harness>")
		assert.Contains(t, stdout, "--local")
		assert.Contains(t, stdout, "--user")
		assert.Contains(t, stdout, "claude")
	}
}

func TestLegacySyncInstructions(t *testing.T) {
	userDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", userDir)
	stdout, stderr, code := runCLI(t, newFakeCatalog(t), "sync-instructions", "claude")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stderr, "deprecated; use gg-lint harness setup")
	assert.Equal(t, "Updated claude user instructions: "+filepath.Join(userDir, "CLAUDE.md")+"\n", stdout)
	assert.FileExists(t, filepath.Join(userDir, "CLAUDE.md"))
}

func TestHarnessRemove(t *testing.T) {
	root := initGitModule(t)
	nested := filepath.Join(root, "nested")
	require.NoError(t, os.Mkdir(nested, 0o755))
	t.Chdir(nested)
	userDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", userDir)
	cat := newFakeCatalog(t)

	userFile := filepath.Join(userDir, "CLAUDE.md")
	localFile := filepath.Join(root, "CLAUDE.md")
	for _, flag := range []string{"--user", "--local"} {
		_, stderr, code := runCLI(t, cat, "harness", "setup", "claude", flag)
		require.Equalf(t, 0, code, "stderr: %s", stderr)
	}

	stdout, stderr, code := runCLI(t, cat, "harness", "remove", "claude")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Equal(t, "Removed claude user instructions: "+userFile+"\n", stdout)
	content, err := os.ReadFile(userFile)
	require.NoError(t, err)
	assert.Empty(t, content)
	content, err = os.ReadFile(localFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), instructions)

	stdout, stderr, code = runCLI(t, cat, "harness", "remove", "claude", "--local")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Equal(t, "Removed claude local instructions: "+localFile+"\n", stdout)
	content, err = os.ReadFile(localFile)
	require.NoError(t, err)
	assert.Empty(t, content)

	stdout, stderr, code = runCLI(t, cat, "harness", "remove", "claude", "--local")
	require.Equalf(t, 0, code, "stderr: %s", stderr)
	assert.Equal(t, "No managed claude local instructions to remove: "+localFile+"\n", stdout)
}
