package harness_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/pkg/harness"
)

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, content, 0o600))
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

func TestAllReturnsHarnessesInNameOrder(t *testing.T) {
	names := make([]string, 0)
	for _, h := range harness.All() {
		names = append(names, h.Name())
	}
	assert.Equal(t, []string{"claude", "codex"}, names)
}

func TestLookupFindsARegisteredHarness(t *testing.T) {
	h, ok := harness.Lookup("codex")
	require.True(t, ok)
	assert.Equal(t, "codex", h.Name())
}

func TestLookupReportsAnUnknownHarness(t *testing.T) {
	_, ok := harness.Lookup("cursor")
	assert.False(t, ok)
}

func TestSyncCreatesOnlyTheSelectedProjectFile(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	codex := filepath.Join(root, "AGENTS.md")
	writeFile(t, claude, []byte("# My Claude notes\n"))
	writeFile(t, codex, []byte("# My Codex notes\n"))

	changed, err := harness.Claude().Sync(root, "run the linter")
	require.NoError(t, err)
	assert.True(t, changed)

	got := string(readFile(t, claude))
	assert.Regexp(t, `^# My Claude notes\n`, got)
	assert.Contains(t, got, "<!-- gg-lint:claude:start -->")
	assert.Contains(t, got, "<!-- gg-lint:claude:end -->")
	assert.Equal(t, "# My Codex notes\n", string(readFile(t, codex)), "unselected AGENTS.md changed")
}

func TestSyncRefreshesTheManagedBlockWithoutChangingSurroundingBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	prefix := []byte("user prefix\r\n")
	suffix := []byte("\r\nuser suffix\x00\n")
	writeFile(t, path, slices.Concat(prefix, []byte("<!-- gg-lint:claude:start -->\r\nold\r\n<!-- gg-lint:claude:end -->"), suffix))

	changed, err := harness.Claude().Sync(root, "run the linter")
	require.NoError(t, err)
	assert.True(t, changed)

	first := readFile(t, path)
	assert.Equal(t, prefix, first[:len(prefix)])
	assert.Equal(t, suffix, first[len(first)-len(suffix):])
	assert.NotContains(t, string(first), "old")

	before, err := os.Stat(path)
	require.NoError(t, err)

	changed, err = harness.Claude().Sync(root, "run the linter")
	require.NoError(t, err)
	assert.False(t, changed, "no-op sync reported a change")

	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), after.ModTime(), "no-op sync wrote the file")
	assert.Equal(t, first, readFile(t, path), "repeat sync was not byte-idempotent")
}

func TestSyncRefusesASymlinkTarget(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "global-instructions.md")
	writeFile(t, outside, []byte("keep these bytes\n"))
	if err := os.Symlink(outside, filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Skipf("cannot create symlink in this environment: %v", err)
	}

	_, err := harness.Claude().Sync(root, "run the linter")
	assert.ErrorContains(t, err, "symlink")
	assert.Equal(t, "keep these bytes\n", string(readFile(t, outside)), "external target changed")
}

func TestSyncCreatesAMissingFile(t *testing.T) {
	root := t.TempDir()

	changed, err := harness.Codex().Sync(root, "run the linter")
	require.NoError(t, err)
	assert.True(t, changed)
	assert.FileExists(t, filepath.Join(root, "AGENTS.md"))
}

func TestSyncReportsAWriteFailureWithoutLeavingAFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not block writes on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission checks")
	}
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o555))
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

	_, err := harness.Claude().Sync(root, "run the linter")
	assert.Error(t, err)
	assert.NoFileExists(t, filepath.Join(root, "CLAUDE.md"))
}

func TestSyncRejectsMalformedMarkersWithoutWriting(t *testing.T) {
	const start = "<!-- gg-lint:claude:start -->"
	const end = "<!-- gg-lint:claude:end -->"
	const codexStart = "<!-- gg-lint:codex:start -->"
	const codexEnd = "<!-- gg-lint:codex:end -->"
	tests := map[string]string{
		"duplicate start": start + "\n" + start + "\n" + end,
		"unmatched start": start + "\nuser content",
		"unmatched end":   end + "\n",
		"reversed":        end + "\n" + start,
		"nested":          start + "\n" + start + "\n" + end + "\n" + end,
		"crossed":         start + "\n" + codexStart + "\n" + end + "\n" + codexEnd,
		"inline markers":  "prefix " + start + "\nold\n" + end + " suffix",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "CLAUDE.md")
			before := []byte("head\n" + content + "\ntail\n")
			writeFile(t, path, before)

			_, err := harness.Claude().Sync(root, "run the linter")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "CLAUDE.md")
			assert.Contains(t, err.Error(), "marker")
			assert.Equal(t, before, readFile(t, path), "malformed file changed")
		})
	}
}

func TestSyncRejectsAForeignHarnessMarker(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "CLAUDE.md")
	before := []byte("head\n<!-- gg-lint:cursor:start -->\nold\n<!-- gg-lint:cursor:end -->\ntail\n")
	writeFile(t, path, before)

	_, err := harness.Claude().Sync(root, "run the linter")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CLAUDE.md")
	assert.Contains(t, err.Error(), "marker")
	assert.Equal(t, before, readFile(t, path), "file with a foreign marker changed")
}
