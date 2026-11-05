package gbp

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// WrapPropagatedErrors flags panic(error) and error wrapping that drops the
// original error.
type WrapPropagatedErrors struct{}

// Meta returns the GBP003 identity, severity, and documentation.
func (WrapPropagatedErrors) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP003",
		Name:    "wrap-propagated-errors",
		Summary: "Flags panic(error) and error wrapping that drops the original error.",
		Description: "Reports `panic` calls with an error argument, and reports `fmt.Errorf` calls that " +
			"pass an error value, or a string carrying only an error's message (`err.Error()`, " +
			"`fmt.Sprint(err)`, or a local variable assigned from them), to an argument whose verb is " +
			"not `%w`. Panicking on an error discards the ability to recover from it, and dropping " +
			"`%w` breaks `errors.Is`/`errors.As` chains downstream. Return a wrapped error with " +
			"`%w` instead. String tracking stays within one function: a helper that returns " +
			"`err.Error()` is not followed.",
		Severity:   lint.SeverityError,
		NeedsTypes: true,
	}
}

// constantString returns the value of expr when it is a constant string.
func constantString(pass *lint.Pass, expr ast.Expr) (string, bool) {
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}

// hasWrapVerb reports whether format contains a real %w verb. It skips %%
// escapes and the usual flags, width, precision, and argument indexes.
func hasWrapVerb(format string) bool {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			i++
			continue
		}
		j := i + 1
		for j < len(format) && strings.ContainsRune("+-# 0123456789.[]*", rune(format[j])) {
			j++
		}
		if j < len(format) && format[j] == 'w' {
			return true
		}
		if j > i+1 {
			i = j - 1
		}
	}
	return false
}

// isStringType reports whether t is the predeclared string type.
func isStringType(t types.Type) bool {
	return t != nil && types.Identical(types.Unalias(t), types.Typ[types.String])
}

// errorStringCall reports whether call is a no-argument Error method call on an
// error value that yields a string, as in err.Error().
func errorStringCall(pass *lint.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Error" {
		return false
	}
	if !isErrorType(pass.TypesInfo.TypeOf(sel.X)) {
		return false
	}
	return isStringType(pass.TypesInfo.TypeOf(call))
}

// consumesError reports whether expr is a string that carries only an error's
// message text, so passing it to fmt.Errorf cannot preserve the error chain.
func consumesError(pass *lint.Pass, expr ast.Expr, tainted map[types.Object]bool) bool {
	if !isStringType(pass.TypesInfo.TypeOf(expr)) {
		return false
	}
	switch x := expr.(type) {
	case *ast.Ident:
		obj := pass.TypesInfo.ObjectOf(x)
		return obj != nil && tainted[obj]
	case *ast.ParenExpr:
		return consumesError(pass, x.X, tainted)
	case *ast.BinaryExpr:
		return x.Op == token.ADD &&
			(consumesError(pass, x.X, tainted) || consumesError(pass, x.Y, tainted))
	case *ast.CallExpr:
		if errorStringCall(pass, x) {
			return true
		}
		if stdlibFunc(pass, x, "fmt", "Sprint") || stdlibFunc(pass, x, "fmt", "Sprintf") {
			for _, arg := range x.Args {
				if isErrorType(pass.TypesInfo.TypeOf(arg)) {
					return true
				}
			}
		}
		// A string helper applied to the error text, such as
		// strings.TrimSpace(err.Error()), carries the same loss.
		for _, arg := range x.Args {
			if consumesError(pass, arg, tainted) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// assignment fills tainted with the object ident denotes when expr carries only
// an error's message text, and reports whether the set grew.
func assignment(pass *lint.Pass, tainted map[types.Object]bool, expr ast.Expr, ident *ast.Ident) bool {
	if !consumesError(pass, expr, tainted) {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil || tainted[obj] {
		return false
	}
	tainted[obj] = true
	return true
}

// taintedStrings returns the objects that a string carrying only an error's
// message text flows into. A local object is unique to its declaration, so one
// set per package cannot confuse two functions.
func taintedStrings(pass *lint.Pass) map[types.Object]bool {
	tainted := make(map[types.Object]bool)
	for {
		changed := false
		visit := func(expr ast.Expr, ident *ast.Ident) {
			if assignment(pass, tainted, expr, ident) {
				changed = true
			}
		}
		record := func(n ast.Node) {
			switch x := n.(type) {
			case *ast.AssignStmt:
				if len(x.Lhs) != len(x.Rhs) {
					return // tuple assignment: positions cannot be paired.
				}
				for i, lhs := range x.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						visit(x.Rhs[i], ident)
					}
				}
			case *ast.ValueSpec:
				if len(x.Names) != len(x.Values) {
					return
				}
				for i, name := range x.Names {
					visit(x.Values[i], name)
				}
			}
		}
		for _, file := range pass.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				record(n)
				return true
			})
		}
		if !changed {
			return tainted
		}
	}
}

// reportsDrop reports whether the fmt.Errorf call at hand drops an error chain
// through one of its arguments.
func reportsDrop(pass *lint.Pass, call *ast.CallExpr, format string, tainted map[types.Object]bool) bool {
	verbs, mapped := argumentVerbs(format)
	if !mapped {
		// The format cannot be paired with arguments, so fall back to the
		// conservative rule: any %w means the author wrapped something.
		if hasWrapVerb(format) {
			return false
		}
		for _, arg := range call.Args[1:] {
			if isErrorType(pass.TypesInfo.TypeOf(arg)) || consumesError(pass, arg, tainted) {
				return true
			}
		}
		return false
	}
	wraps := false
	for i := range call.Args[1:] {
		if verbs[i+1] == 'w' {
			wraps = true
			break
		}
	}
	for i, arg := range call.Args[1:] {
		verb, used := verbs[i+1]
		if !used || verb == 'w' {
			continue // no verb consumes it, or the verb wraps this argument.
		}
		if isErrorType(pass.TypesInfo.TypeOf(arg)) {
			return true // an error dropped from the chain.
		}
		// A stringified error is a stand-in for the original error, so it only
		// signals a loss when the call wraps no error at all. This keeps the
		// recommended "detail: %w" form quiet.
		if !wraps && consumesError(pass, arg, tainted) {
			return true
		}
	}
	return false
}

// Check reports panic(error) calls and unwrapped error propagation in pass.
func (WrapPropagatedErrors) Check(pass *lint.Pass) error {
	tainted := taintedStrings(pass)
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if name, ok := call.Fun.(*ast.Ident); ok && name.Name == "panic" && len(call.Args) > 0 {
				if isErrorType(pass.TypesInfo.TypeOf(call.Args[0])) {
					pass.Reportf(call.Pos(), "avoid panic(error); return a wrapped error")
				}
			}
			if stdlibFunc(pass, call, "fmt", "Errorf") && len(call.Args) > 1 {
				format, ok := constantString(pass, call.Args[0])
				if !ok {
					return true
				}
				if reportsDrop(pass, call, format, tainted) {
					pass.Report(lint.Diagnostic{Pos: call.Pos(), Message: "wrap propagated errors with %w"})
				}
			}
			return true
		})
	}
	return nil
}
