package gbp

import "go/ast"

// callName returns the dotted name of a call expression's function, such as
// "fmt.Errorf" or "panic", or "" when it cannot be determined.
func callName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok {
			return pkg.Name + "." + x.Sel.Name
		}
		return x.Sel.Name
	}
	return ""
}
