package catalog

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

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

var errWrite = errors.New("write failed")

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestWriteTable(t *testing.T) {
	c := renderFixture(t)

	var buf bytes.Buffer
	require.NoError(t, c.WriteTable(&buf))

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], "ID")
	assert.Contains(t, lines[0], "NAME")
	assert.Contains(t, lines[0], "SEVERITY")
	assert.Contains(t, lines[0], "SUMMARY")
	assert.Contains(t, lines[1], "R1")
	assert.Contains(t, lines[1], "no-dot-imports")
	assert.Contains(t, lines[1], "warning")
	assert.Contains(t, lines[1], "flags dot imports")
	assert.Contains(t, lines[2], "R2")
	assert.Contains(t, lines[2], "require-package-doc")
	assert.Contains(t, lines[2], "error")
	assert.Contains(t, lines[2], "requires a package doc comment")
}

func TestWriteMarkdown(t *testing.T) {
	c := renderFixture(t)

	var buf bytes.Buffer
	require.NoError(t, c.WriteMarkdown(&buf))
	got := buf.String()

	assert.Contains(t, got, "# gg-lint rules")
	assert.Contains(t, got, "| ID | Name | Severity | Summary |")

	rules := map[string]struct {
		name     string
		severity string
		summary  string
	}{
		"R1": {"no-dot-imports", "warning", "flags dot imports"},
		"R2": {"require-package-doc", "error", "requires a package doc comment"},
	}
	for id, want := range rules {
		row := "| " + id + " |"
		var line string
		for _, candidate := range strings.Split(got, "\n") {
			if strings.HasPrefix(candidate, row) {
				line = candidate
				break
			}
		}
		require.NotEmpty(t, line, "missing Markdown row for %s", id)
		fields := strings.Split(line, "|")
		require.GreaterOrEqual(t, len(fields), 5)
		assert.Equal(t, id, strings.TrimSpace(fields[1]))
		assert.Equal(t, want.name, strings.TrimSpace(fields[2]))
		assert.Equal(t, want.severity, strings.TrimSpace(fields[3]))
		assert.Equal(t, want.summary, strings.TrimSpace(fields[4]))

		assert.Contains(t, got, "## "+id+": "+want.name)
		assert.Contains(t, got, "Severity: "+want.severity)
	}
	assert.Contains(t, got, "Dot imports pollute the namespace")
	assert.Contains(t, got, "Every package needs a leading doc comment")
}

func TestWriteMarkdownEscapesPipes(t *testing.T) {
	c, err := New(rule("R1", func(m *lint.Meta) {
		m.Name = "pipe"
		m.Summary = "a | b"
		m.Description = "x | y"
	}))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, c.WriteMarkdown(&buf))
	assert.Contains(t, buf.String(), `a \| b`)
}

func TestRenderersPropagateWriteError(t *testing.T) {
	c := renderFixture(t)
	require.ErrorIs(t, c.WriteTable(failWriter{}), errWrite)
	require.ErrorIs(t, c.WriteMarkdown(failWriter{}), errWrite)
}
