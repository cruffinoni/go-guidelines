package gbp

import (
	"go/ast"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// WrapPropagatedErrors flags panic(err) and error wrapping that drops the
// original error.
type WrapPropagatedErrors struct{}

// Meta returns the GBP003 identity, severity, and documentation.
func (WrapPropagatedErrors) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP003",
		Name:    "wrap-propagated-errors",
		Summary: "Flags panic(err) and error wrapping that drops the original error.",
		Description: "Reports `panic(err)` calls whose argument is an identifier named `err`, and reports " +
			"`fmt.Errorf` calls that pass an `err` identifier as an extra argument without a `%w` verb in " +
			"the format string. Panicking on an error discards the ability to recover from it, and " +
			"dropping `%w` breaks `errors.Is`/`errors.As` chains downstream. Return a wrapped error with " +
			"`%w` instead.",
		Severity: lint.SeverityError,
	}
}

// Check reports panic(err) calls and unwrapped error propagation in pass.
func (WrapPropagatedErrors) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := callName(c.Fun)
			if name == "panic" && len(c.Args) > 0 {
				if ident, ok := c.Args[0].(*ast.Ident); ok && ident.Name == "err" {
					pass.Reportf(c.Pos(), "avoid panic(err); return a wrapped error")
				}
			}
			if name == "fmt.Errorf" && len(c.Args) > 1 {
				if lit, ok := c.Args[0].(*ast.BasicLit); ok && !strings.Contains(lit.Value, "%w") {
					for _, arg := range c.Args[1:] {
						if ident, ok := arg.(*ast.Ident); ok && ident.Name == "err" {
							pass.Report(lint.Diagnostic{Pos: c.Pos(), Message: "wrap propagated errors with %w"})
						}
					}
				}
			}
			return true
		})
	}
	return nil
}
