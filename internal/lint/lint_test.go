package lint

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeverityOrderingAndText(t *testing.T) {
	assert.Less(t, SeverityInfo, SeverityWarning)
	assert.Less(t, SeverityWarning, SeverityError)

	for s, want := range map[Severity]string{SeverityInfo: "info", SeverityWarning: "warning", SeverityError: "error"} {
		got, err := s.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, want, string(got))
	}

	_, err := Severity(0).MarshalText()
	assert.Error(t, err)
}

func TestFindingJSONShapeIsStable(t *testing.T) {
	got, err := json.Marshal(Finding{RuleID: "R1", Severity: SeverityError, Message: "m", File: "a.go", Line: 2, Column: 8})
	require.NoError(t, err)
	assert.JSONEq(t, `{"rule_id":"R1","severity":"error","message":"m","file":"a.go","line":2,"column":8}`, string(got))
}

func TestCompareFindingsIsTotalOrder(t *testing.T) {
	ordered := []Finding{
		{File: "a.go", Line: 1, Column: 1, RuleID: "R1", Message: "a"},
		{File: "a.go", Line: 1, Column: 1, RuleID: "R1", Message: "b"},
		{File: "a.go", Line: 1, Column: 1, RuleID: "R2", Message: "a"},
		{File: "a.go", Line: 1, Column: 2, RuleID: "R1", Message: "a"},
		{File: "a.go", Line: 2, Column: 1, RuleID: "R1", Message: "a"},
		{File: "b.go", Line: 1, Column: 1, RuleID: "R1", Message: "a"},
	}
	shuffled := slices.Clone(ordered)
	slices.Reverse(shuffled)
	slices.SortFunc(shuffled, CompareFindings)
	assert.Equal(t, ordered, shuffled)
}

func TestPassExposesSourceAndResolvesFindings(t *testing.T) {
	const filename = "/src/p/a_test.go"
	src := []byte("package p\n\nfunc f() {}\n")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	require.NoError(t, err)

	var got []Diagnostic
	pass := NewPass(fset, []*ast.File{file}, map[string][]byte{filename: src}, func(d Diagnostic) { got = append(got, d) })
	assert.Equal(t, filename, pass.Filename(file))
	assert.True(t, pass.IsTestFile(file))
	assert.Equal(t, src, pass.Source(file))

	fn := file.Decls[0].(*ast.FuncDecl)
	pass.Reportf(fn.Name.Pos(), "bad %s", fn.Name.Name)
	require.Len(t, got, 1)

	finding := NewFinding(Meta{ID: "R1", Severity: SeverityWarning}, fset, got[0])
	assert.Equal(t, Finding{RuleID: "R1", Severity: SeverityWarning, Message: "bad f", File: filename, Line: 3, Column: 6}, finding)
}
