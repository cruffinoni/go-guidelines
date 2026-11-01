package gbp

import (
	"go/ast"
	"go/constant"
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
			"pass an error value to a constant format string without a `%w` verb. Panicking on an error " +
			"discards the ability to recover from it, and dropping `%w` breaks `errors.Is`/`errors.As` " +
			"chains downstream. Return a wrapped error with `%w` instead.",
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

// Check reports panic(error) calls and unwrapped error propagation in pass.
func (WrapPropagatedErrors) Check(pass *lint.Pass) error {
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
				if !ok || hasWrapVerb(format) {
					return true
				}
				for _, arg := range call.Args[1:] {
					if isErrorType(pass.TypesInfo.TypeOf(arg)) {
						pass.Report(lint.Diagnostic{Pos: call.Pos(), Message: "wrap propagated errors with %w"})
						break
					}
				}
			}
			return true
		})
	}
	return nil
}
