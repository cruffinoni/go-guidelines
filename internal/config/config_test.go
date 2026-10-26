package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gg-lint.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadDecodesAValidFile(t *testing.T) {
	path := writeConfig(t, "format = \"json\"\nfail_on = \"warning\"\n[rules]\nTEST001 = false\n")

	file, err := config.Load(path)
	require.NoError(t, err)
	require.NotNil(t, file.Format)
	assert.Equal(t, "json", *file.Format)
	require.NotNil(t, file.FailOn)
	assert.Equal(t, "warning", *file.FailOn)
	assert.Equal(t, map[string]bool{"TEST001": false}, file.Rules)
}

func TestLoadRejectsUnknownKeyWithLineAndColumn(t *testing.T) {
	path := writeConfig(t, "format = \"json\"\nbogus_key = 1\n")

	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path+":2:")
}

func TestLoadRejectsMalformedTOML(t *testing.T) {
	path := writeConfig(t, "format = \nnope\n")

	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}

func TestLoadRejectsDuplicateKey(t *testing.T) {
	path := writeConfig(t, "format = \"json\"\nformat = \"text\"\n")

	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}

func TestLoadReportsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")

	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
}

func TestDiscoverFindsFileAtRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "gg-lint.toml"), []byte("format = \"json\"\n"), 0o600))

	got, err := config.Discover(root)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "gg-lint.toml"), got)
}

func TestDiscoverReturnsEmptyWhenAbsent(t *testing.T) {
	got, err := config.Discover(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}
