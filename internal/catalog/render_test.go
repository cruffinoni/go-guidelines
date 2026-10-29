package catalog

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// update records the golden files from the current render output instead
// of comparing against them, for use after an intentional format change:
// go test ./internal/catalog/... -run Golden -update
var update = flag.Bool("update", false, "update golden files")

func renderFixture(t *testing.T) *Catalog {
	t.Helper()
	c, err := New(
		rule("R1", func(m *lint.Meta) {
			m.Name = "no-dot-imports"
			m.Summary = "flags dot imports"
			m.Description = "Dot imports pollute the namespace and hide where a name comes from."
			m.Severity = lint.SeverityWarning
		}),
		rule("R2", func(m *lint.Meta) {
			m.Name = "require-package-doc"
			m.Summary = "requires a package doc comment"
			m.Description = "Every package needs a leading doc comment describing its purpose."
			m.Severity = lint.SeverityError
		}),
	)
	require.NoError(t, err)
	return c
}

func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, got, 0o644))
		return
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
}

func TestWriteTable_Golden(t *testing.T) {
	c := renderFixture(t)

	var buf bytes.Buffer
	require.NoError(t, c.WriteTable(&buf))

	checkGolden(t, "testdata/table.golden", buf.Bytes())
}

func TestWriteMarkdown_Golden(t *testing.T) {
	c := renderFixture(t)

	var buf bytes.Buffer
	require.NoError(t, c.WriteMarkdown(&buf))

	checkGolden(t, "testdata/markdown.golden", buf.Bytes())
}
