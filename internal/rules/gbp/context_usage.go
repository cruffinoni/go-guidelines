package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ContextUsage flags context.Context stored in a struct field or placed
// after the first parameter.
type ContextUsage struct{}

// Meta returns the GBP004 identity, severity, and documentation.
func (ContextUsage) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP004",
		Name:    "context-usage",
		Summary: "Flags context.Context stored in a struct field or placed after the first parameter.",
		Description: "Reports struct fields typed `context.Context`, and reports function parameters of " +
			"that type that are not the first parameter. Storing a context in a struct hides its lifetime " +
			"and makes its cancellation scope unclear; a context should flow through calls as an explicit, " +
			"leading argument.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports misplaced context.Context usage in pass.
func (ContextUsage) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if st, ok := n.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					if se, ok := field.Type.(*ast.SelectorExpr); ok {
						if pkg, ok := se.X.(*ast.Ident); ok && pkg.Name == "context" && se.Sel.Name == "Context" {
							pass.Reportf(field.Pos(), "do not store context.Context in a struct")
						}
					}
				}
			}
			if ft, ok := n.(*ast.FuncType); ok {
				for i, p := range ft.Params.List {
					if se, ok := p.Type.(*ast.SelectorExpr); ok {
						if pkg, ok := se.X.(*ast.Ident); ok && pkg.Name == "context" && se.Sel.Name == "Context" && i > 0 {
							pass.Reportf(p.Pos(), "context.Context should be the first parameter")
						}
					}
				}
			}
			return true
		})
	}
	return nil
}
