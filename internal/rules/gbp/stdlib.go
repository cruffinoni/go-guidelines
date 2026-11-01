package gbp

import (
	"go/ast"
	"go/types"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// stdlibFunc reports whether call resolves to the named function in the
// standard-library package at path.
func stdlibFunc(pass *lint.Pass, call *ast.CallExpr, path, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok {
		return false
	}
	return obj.Pkg() != nil && obj.Pkg().Path() == path && obj.Name() == name
}

// errorInterface returns the predeclared error interface.
func errorInterface() *types.Interface {
	return types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
}

// isErrorType reports whether t implements error.
func isErrorType(t types.Type) bool {
	return t != nil && types.Implements(t, errorInterface())
}
