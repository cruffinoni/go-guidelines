package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	cat := newFakeCatalog(t)
	stdout, stderr, code := runCLI(t, cat, "version")
	assert.Equal(t, 0, code)
	assert.Empty(t, stderr)
	assert.Equal(t, "gg-lint dev\ncommit: unknown\nbuilt: unknown\n", stdout)
}
