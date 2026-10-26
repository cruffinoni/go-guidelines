package loader_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/loader"
)

// allFiles returns the sorted, deduplicated absolute paths of every Go file
// across pkgs.
func allFiles(pkgs []*loader.Package) []string {
	set := make(map[string]struct{})
	for _, pkg := range pkgs {
		for _, f := range pkg.GoFiles {
			set[f] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}

func abs(t *testing.T, path string) string {
	t.Helper()
	a, err := filepath.Abs(path)
	require.NoError(t, err)
	return a
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
	require.NoError(t, os.WriteFile(name, []byte(content), 0o600))
}

func TestLoad_FilesAndTestVariants(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{Tests: true}, "testdata/mod")
	require.NoError(t, err)

	want := []string{
		abs(t, "testdata/mod/external_test.go"),
		abs(t, "testdata/mod/generated.go"),
		abs(t, "testdata/mod/lib.go"),
		abs(t, "testdata/mod/lib_test.go"),
		abs(t, "testdata/mod/pkga/a.go"),
	}
	slices.Sort(want)
	assert.Equal(t, want, allFiles(pkgs))

	ids := make(map[string]bool, len(pkgs))
	for _, pkg := range pkgs {
		ids[pkg.ID] = true
		assert.False(t, strings.HasSuffix(pkg.ID, ".test") && !strings.Contains(pkg.ID, "["),
			"synthesized test-binary package %q was not dropped", pkg.ID)
	}
	assert.False(t, ids["example.com/fixture"], "plain package should have been superseded by its test variant")
	for _, want := range []string{
		"example.com/fixture [example.com/fixture.test]",
		"example.com/fixture_test [example.com/fixture.test]",
		"example.com/fixture/pkga",
	} {
		assert.True(t, ids[want], "missing expected package %q, got %v", want, ids)
	}
}

func TestLoad_BuildTagExcludesFile(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{Tests: true}, "testdata/mod")
	require.NoError(t, err)
	assert.NotContains(t, allFiles(pkgs), abs(t, "testdata/mod/excluded.go"))
}

func TestLoad_NeedsTypesFalse(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{}, "testdata/mod")
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	for _, pkg := range pkgs {
		assert.Nil(t, pkg.Types, "package %s", pkg.ID)
		assert.Nil(t, pkg.TypesInfo, "package %s", pkg.ID)
	}
}

func TestLoad_NeedsTypesTrue(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{NeedsTypes: true}, "testdata/mod")
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	for _, pkg := range pkgs {
		assert.NotNil(t, pkg.Types, "package %s", pkg.ID)
		assert.NotNil(t, pkg.TypesInfo, "package %s", pkg.ID)
	}
}

func TestLoad_TypeErrorIsKeptNotFatal(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{NeedsTypes: true}, "testdata/typeerr")
	require.NoError(t, err)

	var found bool
	for _, pkg := range pkgs {
		if len(pkg.TypeErrors) > 0 {
			found = true
		}
	}
	assert.True(t, found, "expected at least one package with TypeErrors")
}

func TestLoad_ParseErrorIsFatal(t *testing.T) {
	// Written at test time: a checked-in unparsable file would fail gofmt in CI.
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.test/parseerr\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "bad.go"), "package parseerr\n\nfunc Bad( {\n")

	_, err := loader.Load(context.Background(), loader.Config{}, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad.go")
}

func TestLoad_SingleFile(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{}, "testdata/mod/pkga/a.go")
	require.NoError(t, err)
	assert.Equal(t, []string{abs(t, "testdata/mod/pkga/a.go")}, allFiles(pkgs))
}

func TestLoad_FileOutsideModule(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "foo.go")
	writeFile(t, file, "package foo\n\nfunc Bar() int { return 1 }\n")

	pkgs, err := loader.Load(context.Background(), loader.Config{}, file)
	require.NoError(t, err)
	assert.Contains(t, allFiles(pkgs), file)
	assert.Len(t, pkgs, 1)
}

func TestLoad_DirOutsideModuleIsAnError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "foo.go"), "package foo\n")

	_, err := loader.Load(context.Background(), loader.Config{}, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside any Go module")
}

func TestLoad_FilesOutsideModulesInDifferentDirsAreAllKept(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a", "a.go")
	b := filepath.Join(dir, "b", "b.go")
	writeFile(t, a, "package a\n")
	writeFile(t, b, "package b\n")

	pkgs, err := loader.Load(context.Background(), loader.Config{Tests: true}, a, b)
	require.NoError(t, err)
	assert.Equal(t, []string{a, b}, allFiles(pkgs))
}

func TestLoad_OverlappingInputsDedupe(t *testing.T) {
	dirPkgs, err := loader.Load(context.Background(), loader.Config{}, "testdata/mod/pkga")
	require.NoError(t, err)
	bothPkgs, err := loader.Load(context.Background(), loader.Config{}, "testdata/mod/pkga", "testdata/mod/pkga/a.go")
	require.NoError(t, err)

	assert.Equal(t, allFiles(dirPkgs), allFiles(bothPkgs))
	assert.Len(t, bothPkgs, len(dirPkgs))
}

func TestLoad_SourcesAreExactBytes(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{}, "testdata/mod/pkga/a.go")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Len(t, pkgs[0].GoFiles, 1)

	want, err := os.ReadFile(abs(t, "testdata/mod/pkga/a.go"))
	require.NoError(t, err)
	assert.Equal(t, want, pkgs[0].Sources[pkgs[0].GoFiles[0]])
}

func TestLoad_EmptyPaths(t *testing.T) {
	pkgs, err := loader.Load(context.Background(), loader.Config{})
	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestLoad_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := loader.Load(ctx, loader.Config{}, "testdata/mod")
	require.Error(t, err)
	if !errors.Is(err, context.Canceled) {
		assert.Contains(t, err.Error(), "context canceled")
	}
}
