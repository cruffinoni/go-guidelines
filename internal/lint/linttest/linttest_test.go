package linttest

import (
	"go/ast"
	"path/filepath"
	"strings"
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
	require.Len(t, problems, 4)
	assert.True(t, strings.HasPrefix(problems[0], file+":3:6:"))
	assert.Contains(t, problems[0], "unexpected finding")
	assert.True(t, strings.HasPrefix(problems[1], file+":5:6:"))
	assert.Contains(t, problems[1], "unexpected finding")
	assert.True(t, strings.HasPrefix(problems[2], file+":5:"))
	assert.Contains(t, problems[2], "no finding matching")
	assert.True(t, strings.HasPrefix(problems[3], file+":7:"))
	assert.Contains(t, problems[3], "no finding matching")
}

func TestParsePatternsRejectsMalformedWants(t *testing.T) {
	for _, text := range []string{"", "unquoted", `"("`} {
		_, err := parsePatterns(text)
		assert.Error(t, err, "parsePatterns(%q)", text)
	}
}
