package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// NoPanicInControlFlow flags panic used as normal control flow.
type NoPanicInControlFlow struct{}

// Meta returns the GBP008 identity, severity, and documentation.
func (NoPanicInControlFlow) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP008",
		Name:    "no-panic-in-control-flow",
		Summary: "Flags panic used as normal control flow.",
		Description: "Reports every `panic` call outside test files, including calls already covered by GBP003. " +
			"Panic is meant for unrecoverable programmer errors, not for signaling an expected failure " +
			"condition; return an error instead. Test files are skipped: test fakes panic on purpose, for " +
			"example to exercise recovery.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports every panic call in the non-test files of pass.
func (NoPanicInControlFlow) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		if pass.IsTestFile(file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok && callName(c.Fun) == "panic" {
				pass.Reportf(c.Pos(), "avoid panic in normal control flow")
			}
			return true
		})
	}
	return nil
}
