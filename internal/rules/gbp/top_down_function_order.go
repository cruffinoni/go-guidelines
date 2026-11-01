package gbp

import (
	"go/ast"
	"go/types"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// TopDownFunctionOrder flags functions and methods called before their
// declaration.
type TopDownFunctionOrder struct{}

// Meta returns the GBP021 identity, severity, and documentation.
func (TopDownFunctionOrder) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP021",
		Name:    "top-down-function-order",
		Summary: "Flags functions and methods called before their declaration.",
		Description: "Reports a call to a package-level function or method whose declaration appears " +
			"later in the same file, unless the two declarations are adjacent and mutually recursive. A " +
			"non-recursive forward reference breaks the top-down, declare-before-use reading order; " +
			"reorder the declarations.",
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// orderEntry is one function or method declaration in file order.
type orderEntry struct {
	decl  *ast.FuncDecl
	obj   types.Object
	calls map[types.Object]bool
}

// callObject returns the object of a called function or method, unwrapping
// generic instantiations and parentheses.
func callObject(pass *lint.Pass, expr ast.Expr) types.Object {
	switch x := expr.(type) {
	case *ast.Ident:
		return pass.TypesInfo.Uses[x]
	case *ast.SelectorExpr:
		return pass.TypesInfo.Uses[x.Sel]
	case *ast.IndexExpr:
		return callObject(pass, x.X)
	case *ast.IndexListExpr:
		return callObject(pass, x.X)
	case *ast.ParenExpr:
		return callObject(pass, x.X)
	default:
		return nil
	}
}

// Check reports forward references to later-declared functions and methods
// in pass.
func (TopDownFunctionOrder) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		var entries []orderEntry
		index := make(map[types.Object]int)
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			obj := pass.TypesInfo.Defs[f.Name]
			if obj == nil {
				continue
			}
			index[obj] = len(entries)
			entries = append(entries, orderEntry{decl: f, obj: obj, calls: make(map[types.Object]bool)})
		}

		for i := range entries {
			if entries[i].decl.Body == nil {
				continue
			}
			ast.Inspect(entries[i].decl.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if obj := callObject(pass, call.Fun); obj != nil {
					entries[i].calls[obj] = true
				}
				return true
			})
		}

		for i := range entries {
			if entries[i].decl.Body == nil {
				continue
			}
			ast.Inspect(entries[i].decl.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				obj := callObject(pass, call.Fun)
				j, exists := index[obj]
				if !exists || j <= i {
					return true
				}
				if j == i+1 && entries[j].calls[entries[i].obj] {
					return true
				}
				pass.Reportf(call.Pos(), "function or method is called before its declaration; consider top-down declaration order")
				return true
			})
		}
	}
	return nil
}
