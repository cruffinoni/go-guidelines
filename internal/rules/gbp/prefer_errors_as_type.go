package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// PreferErrorsAsType flags calls to errors.As on Go versions that provide
// errors.AsType.
type PreferErrorsAsType struct{}

// Meta returns the GBP026 identity, severity, and documentation.
func (PreferErrorsAsType) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP026",
		Name:    "prefer-errors-as-type",
		Summary: "Flags errors.As in favor of errors.AsType.",
		Description: "Reports calls to `errors.As`. On Go 1.26 and later, `errors.AsType` avoids the " +
			"pointer indirection of a target variable and checks the target type at compile time; prefer " +
			"`errors.AsType[T](err)`.",
		Severity:   lint.SeverityInfo,
		NeedsTypes: true,
	}
}

// Check reports calls to errors.As in pass.
func (PreferErrorsAsType) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if ok && stdlibFunc(pass, call, "errors", "As") {
				pass.Reportf(call.Pos(), "prefer errors.AsType[T](err) over errors.As")
			}
			return true
		})
	}
	return nil
}
