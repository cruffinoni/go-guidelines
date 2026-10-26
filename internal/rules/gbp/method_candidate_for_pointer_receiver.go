package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// MethodCandidateForPointerReceiver flags functions that take an exported
// struct pointer as their first parameter.
type MethodCandidateForPointerReceiver struct{}

// Meta returns the GBP018 identity, severity, and documentation.
func (MethodCandidateForPointerReceiver) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP018",
		Name:    "method-candidate-for-pointer-receiver",
		Summary: "Flags functions that take an exported struct pointer as their first parameter.",
		Description: "Reports a top-level function whose first parameter is a pointer to an exported " +
			"type. A function that exists only to operate on one type's data is more discoverable, and " +
			"keeps related logic co-located, as a method on that type.",
		Severity: lint.SeverityInfo,
	}
}

// Check reports functions that could be methods in pass.
func (MethodCandidateForPointerReceiver) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok || f.Recv != nil || len(f.Type.Params.List) == 0 {
				return true
			}
			star, ok := f.Type.Params.List[0].Type.(*ast.StarExpr)
			if !ok {
				return true
			}
			if ident, ok := star.X.(*ast.Ident); ok && ast.IsExported(ident.Name) {
				pass.Reportf(f.Pos(), "function taking a struct pointer first may belong as a method")
			}
			return true
		})
	}
	return nil
}
