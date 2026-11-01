package gbp

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// TestIsolation flags test code that mutates process-global state or uses
// hand-rolled test helpers.
type TestIsolation struct{}

// Meta returns the GBP024 identity, severity, and documentation.
func (TestIsolation) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP024",
		Name:    "test-isolation",
		Summary: "Flags tests that mutate process-global state or use legacy helpers.",
		Description: "Reports test-file calls that make tests order-dependent or interfere with parallel " +
			"tests: `os.Setenv`, `os.Unsetenv`, `os.Chdir`, `os.MkdirTemp`, `ioutil.TempDir`, " +
			"`context.Background`/`context.TODO` in test functions, `t.Parallel` combined with " +
			"`t.Setenv`/`t.Chdir`, and `b.N` loops. Use `t.Setenv`, `t.Chdir`, `t.TempDir`, " +
			"`t.Context`, and `b.Loop` instead.",
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// testingType reports whether expr is *testing.T, *testing.B, *testing.F, or
// *testing.M.
func testingType(pass *lint.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "testing" {
		return false
	}
	switch named.Obj().Name() {
	case "T", "B", "F", "M":
		return true
	default:
		return false
	}
}

// testingParam reports whether f declares a *testing.T/B/F/M parameter.
func testingParam(pass *lint.Pass, f *ast.FuncDecl) bool {
	for _, field := range f.Type.Params.List {
		if testingType(pass, field.Type) {
			return true
		}
	}
	return false
}

// testingMethod reports whether call invokes the named testing.T/B/F/M
// method.
func testingMethod(pass *lint.Pass, call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "testing" || obj.Name() != name {
		return false
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	t := sig.Recv().Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "testing" {
		return false
	}
	switch named.Obj().Name() {
	case "T", "B", "F":
		return true
	default:
		return false
	}
}

// benchmarkN reports whether expr is a b.N selector on *testing.B.
func benchmarkN(pass *lint.Pass, expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "N" {
		return false
	}
	t := pass.TypesInfo.TypeOf(sel.X)
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "testing" && named.Obj().Name() == "B"
}

// checkTestFunction reports test-local isolation issues in f.
func checkTestFunction(pass *lint.Pass, f *ast.FuncDecl) {
	if f.Body == nil || !testingParam(pass, f) {
		return
	}
	var envCalls []*ast.CallExpr
	var parallel bool
	ast.Inspect(f.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if testingMethod(pass, call, "Parallel") {
			parallel = true
		}
		if testingMethod(pass, call, "Setenv") || testingMethod(pass, call, "Chdir") {
			envCalls = append(envCalls, call)
		}
		if stdlibFunc(pass, call, "context", "Background") || stdlibFunc(pass, call, "context", "TODO") {
			pass.Reportf(call.Pos(), "use t.Context instead of context.Background or context.TODO in tests")
		}
		return true
	})
	if parallel {
		for _, call := range envCalls {
			pass.Reportf(call.Pos(), "t.Parallel cannot be combined with t.Setenv or t.Chdir")
		}
	}
}

// benchmarkLoop reports whether f is a b.N loop.
func benchmarkLoop(pass *lint.Pass, f *ast.ForStmt) bool {
	cond, ok := f.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.LSS {
		return false
	}
	return benchmarkN(pass, cond.X) || benchmarkN(pass, cond.Y)
}

// benchmarkRange reports whether r ranges over b.N.
func benchmarkRange(pass *lint.Pass, r *ast.RangeStmt) bool {
	return benchmarkN(pass, r.X)
}

// Check reports test-isolation problems in pass.
func (TestIsolation) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		if !pass.IsTestFile(file) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch {
			case stdlibFunc(pass, call, "os", "Setenv"):
				pass.Reportf(call.Pos(), "avoid os.Setenv in tests; use t.Setenv")
			case stdlibFunc(pass, call, "os", "Unsetenv"):
				pass.Reportf(call.Pos(), "avoid os.Unsetenv in tests; restore state with t.Cleanup")
			case stdlibFunc(pass, call, "os", "Chdir"):
				pass.Reportf(call.Pos(), "avoid os.Chdir in tests; use t.Chdir")
			case stdlibFunc(pass, call, "os", "MkdirTemp"):
				pass.Reportf(call.Pos(), "avoid os.MkdirTemp in tests; use t.TempDir or t.ArtifactDir")
			case stdlibFunc(pass, call, "io/ioutil", "TempDir"):
				pass.Reportf(call.Pos(), "avoid ioutil.TempDir in tests; use t.TempDir or t.ArtifactDir")
			}
			return true
		})

		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				checkTestFunction(pass, x)
			case *ast.ForStmt:
				if benchmarkLoop(pass, x) {
					pass.Reportf(x.Pos(), "use b.Loop instead of a b.N loop")
				}
			case *ast.RangeStmt:
				if benchmarkRange(pass, x) {
					pass.Reportf(x.Pos(), "use b.Loop instead of a b.N loop")
				}
			}
			return true
		})
	}
	return nil
}
