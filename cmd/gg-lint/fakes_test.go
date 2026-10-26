package main

import (
	"bytes"
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// identRule reports every occurrence of an identifier named want. It is a
// fake catalog rule used only by command tests, so tests never couple to
// real GBP/GCM rule IDs.
type identRule struct {
	meta lint.Meta
	want string
}

func (r identRule) Meta() lint.Meta { return r.meta }

// Check reports every identifier named r.want in pass.
func (r identRule) Check(pass *lint.Pass) error {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == r.want {
				pass.Reportf(id.Pos(), "found identifier %s", r.want)
			}
			return true
		})
	}
	return nil
}

// newFakeCatalog returns a catalog with two fake rules: TEST001 (error
// severity) flags the identifier Bad, and TEST002 (info severity) flags
// the identifier Watch.
func newFakeCatalog(t testing.TB) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.New(
		identRule{want: "Bad", meta: lint.Meta{
			ID: "TEST001", Name: "test-bad-ident",
			Summary: "flags the identifier Bad.", Description: "Flags the identifier Bad, for command tests.",
			Severity: lint.SeverityError,
		}},
		identRule{want: "Watch", meta: lint.Meta{
			ID: "TEST002", Name: "test-watch-ident",
			Summary: "flags the identifier Watch.", Description: "Flags the identifier Watch, for command tests.",
			Severity: lint.SeverityInfo,
		}},
	)
	require.NoError(t, err)
	return cat
}

// writeModule creates a Go module in a new temporary directory with files,
// changes the working directory to it for the duration of the test, and
// returns its path.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/x\n\ngo 1.27\n"), 0o644))
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	t.Chdir(dir)
	return dir
}

// runCLI runs gg-lint with cat and args, and returns stdout, stderr, and
// the resulting exit code.
func runCLI(t testing.TB, cat *catalog.Catalog, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	code = run(context.Background(), args, &outBuf, &errBuf, cat)
	return outBuf.String(), errBuf.String(), code
}
