package gbp

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// NoNilCheckForRequiredDependency flags initializers that defensively
// nil-check a required dependency.
type NoNilCheckForRequiredDependency struct{}

// Meta returns the GBP019 identity, severity, and documentation.
func (NoNilCheckForRequiredDependency) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP019",
		Name:    "no-nil-check-for-required-dependency",
		Summary: "Flags initializers that defensively nil-check a required dependency.",
		Description: "Reports nil checks on constructor or initializer parameters, and on unexported " +
			"fields of those parameters, when the nil branch silently returns a zero value or skips " +
			"initialization. Missing required dependencies should fail immediately; fail-fast checks " +
			"that return an error or panic are allowed.",
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// initializerPrefixes are the naming conventions used to recognize
// constructors and initializers.
var initializerPrefixes = []string{"New", "new", "Open", "open", "Build", "build", "Init"}

// isInitializerName reports whether name follows an initializer convention.
func isInitializerName(name string) bool {
	for _, prefix := range initializerPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// dependencyParams is the set of parameter and receiver objects in f, split
// into dependencies and error-valued parameters.
type dependencyParams struct {
	dependencies map[types.Object]bool
	errors       map[types.Object]bool
}

// constructorParams collects the parameter and receiver objects of f.
func constructorParams(pass *lint.Pass, f *ast.FuncDecl) dependencyParams {
	params := dependencyParams{
		dependencies: make(map[types.Object]bool),
		errors:       make(map[types.Object]bool),
	}
	add := func(field *ast.FieldList) {
		if field == nil {
			return
		}
		for _, param := range field.List {
			for _, name := range param.Names {
				obj := pass.TypesInfo.Defs[name]
				if obj == nil {
					continue
				}
				params.dependencies[obj] = true
				if isErrorType(obj.Type()) {
					params.errors[obj] = true
				}
			}
		}
	}
	add(f.Type.Params)
	if f.Recv != nil {
		add(f.Recv)
	}
	return params
}

// isNilIdent reports whether expr is the predeclared nil identifier.
func isNilIdent(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "nil"
}

// nilComparedExpr returns the non-nil operand of a nil comparison.
func nilComparedExpr(b *ast.BinaryExpr) ast.Expr {
	if isNilIdent(b.Y) {
		return b.X
	}
	if isNilIdent(b.X) {
		return b.Y
	}
	return nil
}

// dependencyExpr reports whether expr is a required dependency: a parameter,
// receiver, or a field selected from one.
func dependencyExpr(pass *lint.Pass, expr ast.Expr, params dependencyParams) bool {
	switch x := expr.(type) {
	case *ast.Ident:
		obj := pass.TypesInfo.Uses[x]
		return obj != nil && params.dependencies[obj] && !params.errors[obj]
	case *ast.SelectorExpr:
		base, ok := x.X.(*ast.Ident)
		if !ok {
			return false
		}
		obj := pass.TypesInfo.Uses[base]
		if obj == nil || !params.dependencies[obj] || params.errors[obj] {
			return false
		}
		selection := pass.TypesInfo.Selections[x]
		if selection == nil || selection.Kind() != types.FieldVal {
			return false
		}
		field, ok := pass.TypesInfo.Uses[x.Sel].(*types.Var)
		return ok && !field.Exported()
	default:
		return false
	}
}

// returnsError reports whether ret returns a non-nil error in a result
// position whose type implements error.
func returnsError(pass *lint.Pass, f *ast.FuncDecl, ret *ast.ReturnStmt) bool {
	obj, ok := pass.TypesInfo.Defs[f.Name].(*types.Func)
	if !ok {
		return false
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return false
	}
	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		if !isErrorType(results.At(i).Type()) {
			continue
		}
		if i >= len(ret.Results) || isNilIdent(ret.Results[i]) {
			continue
		}
		if isErrorType(pass.TypesInfo.TypeOf(ret.Results[i])) {
			return true
		}
	}
	return false
}

// failFast reports whether block returns an error, or panics, immediately.
func failFast(pass *lint.Pass, f *ast.FuncDecl, block *ast.BlockStmt) bool {
	if block == nil {
		return false
	}
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		switch x := n.(type) {
		case *ast.CallExpr:
			if name, ok := x.Fun.(*ast.Ident); ok && name.Name == "panic" {
				found = true
				return false
			}
		case *ast.ReturnStmt:
			if returnsError(pass, f, x) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// elseFailsFast reports whether an else branch fails immediately.
func elseFailsFast(pass *lint.Pass, f *ast.FuncDecl, stmt ast.Stmt) bool {
	switch x := stmt.(type) {
	case *ast.BlockStmt:
		return failFast(pass, f, x)
	case *ast.IfStmt:
		return failFast(pass, f, x.Body)
	default:
		return false
	}
}

// Check reports defensive nil checks in initializers in pass.
func (NoNilCheckForRequiredDependency) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || f.Body == nil || !isInitializerName(f.Name.Name) {
				continue
			}
			params := constructorParams(pass, f)
			ast.Inspect(f.Body, func(n ast.Node) bool {
				s, ok := n.(*ast.IfStmt)
				if !ok {
					return true
				}
				found := false
				ast.Inspect(s.Cond, func(node ast.Node) bool {
					if found {
						return false
					}
					b, ok := node.(*ast.BinaryExpr)
					if !ok || (b.Op != token.EQL && b.Op != token.NEQ) {
						return true
					}
					expr := nilComparedExpr(b)
					if expr == nil || !dependencyExpr(pass, expr, params) {
						return true
					}
					if b.Op == token.EQL && failFast(pass, f, s.Body) {
						return true
					}
					if b.Op == token.NEQ && elseFailsFast(pass, f, s.Else) {
						return true
					}
					pass.Reportf(s.Pos(), "initializer nil-checks a required dependency instead of failing fast")
					found = true
					return false
				})
				return true
			})
		}
	}
	return nil
}
