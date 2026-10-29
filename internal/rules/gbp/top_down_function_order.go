package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// TopDownFunctionOrder flags local functions called before their
// declaration.
type TopDownFunctionOrder struct{}

// Meta returns the GBP021 identity, severity, and documentation.
func (TopDownFunctionOrder) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP021",
		Name:    "top-down-function-order",
		Summary: "Flags local functions called before their declaration.",
		Description: "Reports a call to a package-level function whose declaration appears later in the " +
			"file, unless the two functions are adjacent and mutually recursive. A non-recursive forward " +
			"reference breaks the top-down, declare-before-use reading order; reorder the declarations.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports forward references to later-declared functions in pass.
func (TopDownFunctionOrder) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		var funcs []*ast.FuncDecl
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil {
				funcs = append(funcs, f)
			}
		}
		index := map[string]int{}
		for i, f := range funcs {
			if _, exists := index[f.Name.Name]; !exists {
				index[f.Name.Name] = i
			}
		}
		calls := make([]map[string]bool, len(funcs))
		for i, f := range funcs {
			calls[i] = make(map[string]bool)
			if f.Body != nil {
				ast.Inspect(f.Body, func(n ast.Node) bool {
					if c, ok := n.(*ast.CallExpr); ok {
						if called, ok := c.Fun.(*ast.Ident); ok {
							calls[i][called.Name] = true
						}
					}
					return true
				})
			}
		}
		for i, f := range funcs {
			if f.Body == nil {
				continue
			}
			ast.Inspect(f.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				id, ok := c.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				if j, exists := index[id.Name]; exists && j > i && j != i {
					if j == i+1 && calls[j][f.Name.Name] {
						return true
					}
					pass.Reportf(c.Pos(), "local function is called before its declaration; consider top-down declaration order")
				}
				return true
			})
		}
	}
	return nil
}
