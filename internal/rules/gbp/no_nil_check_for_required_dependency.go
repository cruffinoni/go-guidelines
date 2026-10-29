package gbp

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// NoNilCheckForRequiredDependency flags constructors that nil-check a
// required dependency.
type NoNilCheckForRequiredDependency struct{}

// Meta returns the GBP019 identity, severity, and documentation.
func (NoNilCheckForRequiredDependency) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP019",
		Name:    "no-nil-check-for-required-dependency",
		Summary: "Flags constructors that nil-check a required dependency.",
		Description: "In a function whose name starts with `New`, reports an `if x == nil` check on a " +
			"non-`err` parameter. Nil-checking a required dependency in a constructor hides wiring bugs " +
			"behind an optional-looking code path instead of failing fast; enforce the dependency as part " +
			"of the constructor's contract.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports constructors that nil-check a required dependency in pass.
func (NoNilCheckForRequiredDependency) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(f.Name.Name, "New") || f.Body == nil {
				return true
			}
			ast.Inspect(f.Body, func(child ast.Node) bool {
				if s, ok := child.(*ast.IfStmt); ok {
					if b, ok := s.Cond.(*ast.BinaryExpr); ok && b.Op == token.EQL {
						if id, ok := b.Y.(*ast.Ident); ok && id.Name == "nil" {
							if x, ok := b.X.(*ast.Ident); ok && x.Name != "err" {
								pass.Reportf(s.Pos(), "constructor nil-checks a required dependency instead of enforcing its contract")
							}
						}
					}
				}
				return true
			})
			return true
		})
	}
	return nil
}
