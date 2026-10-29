package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// PreferNilSlice flags empty slice literals in favor of nil slices.
type PreferNilSlice struct{}

// Meta returns the GBP009 identity, severity, and documentation.
func (PreferNilSlice) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP009",
		Name:    "prefer-nil-slice",
		Summary: "Flags empty slice literals in favor of nil slices.",
		Description: "Reports composite literals such as `[]T{}` that have no elements and no explicit " +
			"length. A nil slice behaves identically to an empty one for almost every operation, so " +
			"`var s []T` is simpler. When a non-nil empty slice is required, such as to encode `[]` " +
			"instead of `null` in JSON, write `make([]T, 0)` to make that intent explicit.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports empty slice literals in pass.
func (PreferNilSlice) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CompositeLit); ok && len(c.Elts) == 0 {
				if a, ok := c.Type.(*ast.ArrayType); ok && a.Len == nil {
					pass.Reportf(c.Pos(), "empty slice literal; prefer nil slice unless non-nil JSON semantics are required")
				}
			}
			return true
		})
	}
	return nil
}
