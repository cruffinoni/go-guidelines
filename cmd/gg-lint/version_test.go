package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	oldVersion, oldCommit, oldBuiltAt := version, commit, builtAt
	t.Cleanup(func() {
		version, commit, builtAt = oldVersion, oldCommit, oldBuiltAt
	})
	version, commit, builtAt = "1.2.3", "abc123", "2025-01-02T03:04:05Z"

	cat := newFakeCatalog(t)
	stdout, stderr, code := runCLI(t, cat, "version")
	assert.Equal(t, 0, code)
	assert.Empty(t, stderr)

	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], version)
	assert.Contains(t, lines[1], commit)
	assert.Contains(t, lines[2], builtAt)
}
