package scope_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/scope"
)

// writeGoFile creates a minimal, valid-looking Go file at path; its content
// does not matter to scope, which never parses files.
func writeGoFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("package p\n"), 0o644))
}

func TestNew_RejectsEmptyPaths(t *testing.T) {
	_, err := scope.New(nil, nil, nil)
	assert.Error(t, err)
}

func TestNew_RejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	_, err := scope.New([]string{filepath.Join(dir, "missing.go")}, nil, nil)
	assert.ErrorContains(t, err, "inspect")
}

func TestNew_RejectsNonGoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	_, err := scope.New([]string{path}, nil, nil)
	assert.ErrorContains(t, err, "not a Go source file or directory")
}

func TestSet_Paths_ReturnsInputsInOrder(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.go")
	b := filepath.Join(dir, "sub")
	writeGoFile(t, a)
	require.NoError(t, os.MkdirAll(b, 0o755))

	s, err := scope.New([]string{a, b}, nil, nil)
	require.NoError(t, err)

	got := s.Paths()
	require.Len(t, got, 2)
	assert.True(t, filepath.IsAbs(got[0]))
	assert.True(t, filepath.IsAbs(got[1]))
}

func TestSet_Contains_FileInput(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.go")
	other := filepath.Join(dir, "b.go")
	writeGoFile(t, a)
	writeGoFile(t, other)

	s, err := scope.New([]string{a}, nil, nil)
	require.NoError(t, err)

	assert.True(t, s.Contains(a))
	assert.False(t, s.Contains(other))
}

func TestSet_Contains_DirectoryInput(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "pkg", "a.go")
	outside := filepath.Join(filepath.Dir(dir), "elsewhere.go")
	writeGoFile(t, inner)
	writeGoFile(t, outside)

	s, err := scope.New([]string{dir}, nil, nil)
	require.NoError(t, err)

	assert.True(t, s.Contains(inner))
	assert.False(t, s.Contains(outside))
}

func TestSet_Contains_SegmentPrefixNotStringPrefix(t *testing.T) {
	dir := t.TempDir()
	b := filepath.Join(dir, "b")
	bc := filepath.Join(dir, "bc", "f.go")
	writeGoFile(t, filepath.Join(b, "f.go"))
	writeGoFile(t, bc)

	s, err := scope.New([]string{b}, nil, nil)
	require.NoError(t, err)

	assert.False(t, s.Contains(bc), "/bc must not match an input of /b")
}

func TestSet_Contains_Include(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "keep", "a.go")
	dropped := filepath.Join(dir, "drop", "b.go")
	writeGoFile(t, kept)
	writeGoFile(t, dropped)

	s, err := scope.New([]string{dir}, []string{filepath.Join(dir, "keep")}, nil)
	require.NoError(t, err)

	assert.True(t, s.Contains(kept))
	assert.False(t, s.Contains(dropped))
}

func TestSet_Contains_Exclude(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "keep", "a.go")
	dropped := filepath.Join(dir, "drop", "b.go")
	writeGoFile(t, kept)
	writeGoFile(t, dropped)

	s, err := scope.New([]string{dir}, nil, []string{filepath.Join(dir, "drop")})
	require.NoError(t, err)

	assert.True(t, s.Contains(kept))
	assert.False(t, s.Contains(dropped))
}

func TestSet_Contains_IncludeAndExcludeCombine(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "keep", "a.go")
	alsoInInclude := filepath.Join(dir, "keep", "sub", "b.go")
	writeGoFile(t, kept)
	writeGoFile(t, alsoInInclude)

	s, err := scope.New(
		[]string{dir},
		[]string{filepath.Join(dir, "keep")},
		[]string{filepath.Join(dir, "keep", "sub")},
	)
	require.NoError(t, err)

	assert.True(t, s.Contains(kept))
	assert.False(t, s.Contains(alsoInInclude))
}

// IncludeExclude selectors need not exist on disk.
func TestSet_Contains_SelectorsNeedNotExist(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "a.go")
	writeGoFile(t, kept)

	s, err := scope.New([]string{dir}, []string{filepath.Join(dir, "does-not-exist")}, nil)
	require.NoError(t, err)

	assert.False(t, s.Contains(kept))
}

func TestSet_Contains_SymlinkedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}
	real := t.TempDir()
	file := filepath.Join(real, "a.go")
	writeGoFile(t, file)

	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(real, link))

	s, err := scope.New([]string{link}, nil, nil)
	require.NoError(t, err)

	assert.True(t, s.Contains(file))
	assert.True(t, s.Contains(filepath.Join(link, "a.go")))
}
