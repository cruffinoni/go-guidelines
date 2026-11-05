package config_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), config.FileName)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// writeUserConfig points XDG_CONFIG_HOME at a fresh directory and writes
// the user config file there, returning its path.
func writeUserConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "gg-lint", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("format: json\n"), 0o600))
	return path
}

func TestLoadDecodesAValidFile(t *testing.T) {
	path := writeConfig(t, "format: json\nfail_on: warning\nrules:\n  TEST001: false\n")

	file, err := config.Load(path)
	require.NoError(t, err)
	require.NotNil(t, file.Format)
	assert.Equal(t, "json", *file.Format)
	require.NotNil(t, file.FailOn)
	assert.Equal(t, "warning", *file.FailOn)
	assert.Equal(t, map[string]bool{"TEST001": false}, file.Rules)
}

func TestLoadDecodesAnEmptyFile(t *testing.T) {
	path := writeConfig(t, "")

	file, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, config.File{}, file)
}

func TestLoadRejectsInvalidFiles(t *testing.T) {
	tests := map[string]struct {
		content string
		missing bool
		wantErr error
	}{
		"multiple documents": {
			content: "format: json\n---\nbogus_key: true\n",
			wantErr: config.ErrMultipleDocuments,
		},
		"unknown key": {
			content: "format: json\nbogus_key: 1\n",
			wantErr: config.ErrDecode,
		},
		"malformed yaml": {
			content: "format: [json\n",
			wantErr: config.ErrDecode,
		},
		"duplicate key": {
			content: "format: json\nformat: text\n",
			wantErr: config.ErrDecode,
		},
		"missing file": {
			missing: true,
			wantErr: fs.ErrNotExist,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var path string
			if tc.missing {
				path = filepath.Join(t.TempDir(), "missing.yaml")
			} else {
				path = writeConfig(t, tc.content)
			}

			_, err := config.Load(path)
			require.Error(t, err)
			require.ErrorIs(t, err, tc.wantErr)
			fileErr, ok := errors.AsType[*config.FileError](err)
			require.True(t, ok)
			assert.Equal(t, path, fileErr.Path)
		})
	}
}

func TestUserPathHonorsXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got, err := config.UserPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "gg-lint", "config.yaml"), got)
}

func TestUserPathFallsBackToDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir uses USERPROFILE on Windows

	got, err := config.UserPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "gg-lint", "config.yaml"), got)
}

func TestDiscoverPrefersProjectFile(t *testing.T) {
	writeUserConfig(t)
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, config.FileName), []byte("format: json\n"), 0o600))

	got, err := config.Discover(root)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, config.FileName), got)
}

func TestDiscoverFallsBackToUserFile(t *testing.T) {
	userPath := writeUserConfig(t)

	got, err := config.Discover(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, userPath, got)
}

func TestDiscoverWithoutRootChecksOnlyUserFile(t *testing.T) {
	userPath := writeUserConfig(t)

	got, err := config.Discover("")
	require.NoError(t, err)
	assert.Equal(t, userPath, got)
}

func TestDiscoverReturnsEmptyWhenAbsent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	got, err := config.Discover(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}
