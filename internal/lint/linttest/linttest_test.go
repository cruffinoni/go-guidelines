package linttest

import (
	"go/ast"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// funcNames reports the name of every top-level function.
type funcNames struct{}

func (funcNames) Meta() lint.Meta {
	return lint.Meta{ID: "TEST001", Name: "func-names", Severity: lint.SeverityInfo}
}

func (funcNames) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				pass.Reportf(fn.Name.Pos(), "func %s", fn.Name.Name)
			}
		}
	}
	return nil
}

func TestRunAcceptsMatchingFindingsAcrossPackages(t *testing.T) {
	Run(t, funcNames{}, "testdata/ok")
}

func TestCheckReportsEveryMismatch(t *testing.T) {
	problems, err := check(funcNames{}, "testdata/mismatch")
	require.NoError(t, err)
	file := filepath.Join("testdata", "mismatch", "a.go")
	want := []string{
		file + `:3:6: unexpected finding "func unexpected"`,
		file + `:5:6: unexpected finding "func wrongMessage"`,
		file + `:5: no finding matching "something else"`,
		file + `:7: no finding matching "never reported"`,
	}
	assert.Equal(t, want, problems)
}

func TestParsePatternsRejectsMalformedWants(t *testing.T) {
	for _, text := range []string{"", "unquoted", `"("`} {
		_, err := parsePatterns(text)
		assert.Error(t, err, "parsePatterns(%q)", text)
	}
}
