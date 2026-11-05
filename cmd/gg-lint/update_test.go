package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateFlagsRegistered(t *testing.T) {
	cmd := newUpdateCommand()
	for _, name := range []string{"version", "bin-dir", "no-path-update", "quiet"} {
		assert.NotNil(t, cmd.Flags().Lookup(name), "flag --%s is not registered", name)
	}
}

func TestUpdateInvalidVersion(t *testing.T) {
	cat := newFakeCatalog(t)
	stdout, stderr, code := runCLI(t, cat, "update", "--version=not-a-version")
	assert.Equal(t, 2, code)
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}

func TestUpdateVersionFromEnvironment(t *testing.T) {
	t.Setenv("GG_LINT_VERSION", "not-a-version")
	cat := newFakeCatalog(t)
	stdout, stderr, code := runCLI(t, cat, "update")
	assert.Equal(t, 2, code)
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}

func TestUpdateRejectsArguments(t *testing.T) {
	cat := newFakeCatalog(t)
	_, stderr, code := runCLI(t, cat, "update", "extra")
	assert.Equal(t, 2, code)
	assert.NotEmpty(t, stderr)
}
