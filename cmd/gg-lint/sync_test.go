package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncInstructions_UnknownHarness(t *testing.T) {
	cat := newFakeCatalog(t)
	stdout, stderr, code := runCLI(t, cat, "sync-instructions", "--harness=nope")
	assert.Equal(t, 2, code)
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}
