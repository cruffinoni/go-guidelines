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

// isStringType reports whether t has string as its underlying type.
func isStringType(t types.Type) bool {
	if t == nil {
		return false
	}
	basic, ok := types.Unalias(t).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}

type errorSources struct {
	objects map[types.Object]bool
	unknown bool
}

type stringState map[types.Object]errorSources

func sourceOfError(pass *lint.Pass, expr ast.Expr) errorSources {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
			return errorSources{objects: map[types.Object]bool{obj: true}}
		}
	}
	return errorSources{unknown: true}
}

func mergeSources(dst *errorSources, src errorSources) {
	if dst.objects == nil {
		dst.objects = make(map[types.Object]bool)
	}
	for obj := range src.objects {
		dst.objects[obj] = true
	}
	dst.unknown = dst.unknown || src.unknown
}

func stringSources(pass *lint.Pass, expr ast.Expr, state stringState) errorSources {
	if !isStringType(pass.TypesInfo.TypeOf(expr)) {
		return errorSources{}
	}
	switch x := expr.(type) {
	case *ast.Ident:
		if obj := pass.TypesInfo.ObjectOf(x); obj != nil {
			return state[obj]
		}
	case *ast.ParenExpr:
		return stringSources(pass, x.X, state)
	case *ast.BinaryExpr:
		if x.Op == token.ADD {
			out := errorSources{}
			mergeSources(&out, stringSources(pass, x.X, state))
			mergeSources(&out, stringSources(pass, x.Y, state))
			return out
		}
	case *ast.CallExpr:
		if errorStringCall(pass, x) {
			sel := x.Fun.(*ast.SelectorExpr)
			return sourceOfError(pass, sel.X)
		}
		if stdlibFunc(pass, x, "fmt", "Sprint") || stdlibFunc(pass, x, "fmt", "Sprintf") {
			out := errorSources{}
			for _, arg := range x.Args {
				if isErrorType(pass.TypesInfo.TypeOf(arg)) {
					mergeSources(&out, sourceOfError(pass, arg))
				} else {
					mergeSources(&out, stringSources(pass, arg, state))
				}
			}
			return out
		}
		// Propagate only through known string-preserving standard-library helpers.
		if isStringHelper(pass, x) {
			out := errorSources{}
			for _, arg := range x.Args {
				mergeSources(&out, stringSources(pass, arg, state))
			}
			return out
		}
	}
	return errorSources{}
}

func isStringHelper(pass *lint.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	for _, name := range []string{
		"TrimSpace", "Trim", "TrimLeft", "TrimRight", "TrimPrefix", "TrimSuffix",
		"ToLower", "ToUpper", "ToTitle", "Replace", "ReplaceAll", "Map",
		"Repeat", "Clone",
	} {
		if stdlibFunc(pass, call, "strings", name) {
			return true
		}
	}
	return false
}

func errorStringCall(pass *lint.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Error" || !isErrorType(pass.TypesInfo.TypeOf(sel.X)) {
		return false
	}
	return isStringType(pass.TypesInfo.TypeOf(call))
}

func cloneStringState(state stringState) stringState {
	out := make(stringState, len(state))
	for obj, sources := range state {
		copy := errorSources{unknown: sources.unknown}
		mergeSources(&copy, sources)
		out[obj] = copy
	}
	return out
}

func mergeStringStates(states ...stringState) stringState {
	out := make(stringState)
	for _, state := range states {
		for obj, sources := range state {
			merged := out[obj]
			mergeSources(&merged, sources)
			out[obj] = merged
		}
	}
	return out
}

func assignString(pass *lint.Pass, state stringState, expr ast.Expr, ident *ast.Ident) {
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return
	}
	sources := stringSources(pass, expr, state)
	if len(sources.objects) == 0 && !sources.unknown {
		delete(state, obj)
		return
	}
	state[obj] = sources
}

func sourceObject(pass *lint.Pass, expr ast.Expr) (types.Object, bool) {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil, false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	return obj, obj != nil
}

func reportsDrop(pass *lint.Pass, call *ast.CallExpr, format string, state stringState) bool {
	verbs, mapped := argumentVerbs(format)
	if !mapped {
		if hasWrapVerb(format) {
			return false
		}
		for _, arg := range call.Args[1:] {
			if isErrorType(pass.TypesInfo.TypeOf(arg)) {
				return true
			}
			sources := stringSources(pass, arg, state)
			if len(sources.objects) > 0 || sources.unknown {
				return true
			}
		}
		return false
	}

	wrapped := make(map[types.Object]bool)
	wrapsUnknown := false
	for i, arg := range call.Args[1:] {
		if verbs[i+1] != 'w' {
			continue
		}
		if obj, ok := sourceObject(pass, arg); ok {
			wrapped[obj] = true
		} else {
			wrapsUnknown = true
		}
	}

	for i, arg := range call.Args[1:] {
		verb, used := verbs[i+1]
		if !used || verb == 'w' {
			continue
		}
		if isErrorType(pass.TypesInfo.TypeOf(arg)) {
			return true
		}
		sources := stringSources(pass, arg, state)
		if sources.unknown && !wrapsUnknown {
			return true
		}
		for obj := range sources.objects {
			if !wrapped[obj] {
				return true
			}
		}
	}
	return false
}

func inspectExpr(pass *lint.Pass, expr ast.Expr, state stringState) {
	ast.Inspect(expr, func(n ast.Node) bool {
		if _, isLiteral := n.(*ast.FuncLit); isLiteral {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if name, ok := call.Fun.(*ast.Ident); ok && name.Name == "panic" && len(call.Args) > 0 &&
			isErrorType(pass.TypesInfo.TypeOf(call.Args[0])) {
			pass.Reportf(call.Pos(), "avoid panic(error); return a wrapped error")
		}
		if stdlibFunc(pass, call, "fmt", "Errorf") && len(call.Args) > 1 {
			if format, ok := constantString(pass, call.Args[0]); ok && reportsDrop(pass, call, format, state) {
				pass.Report(lint.Diagnostic{Pos: call.Pos(), Message: "wrap propagated errors with %w"})
			}
		}
		return true
	})
}

func processStmt(pass *lint.Pass, stmt ast.Stmt, state stringState) stringState {
	switch x := stmt.(type) {
	case *ast.BlockStmt:
		return processBlock(pass, x, state)
	case *ast.AssignStmt:
		for _, rhs := range x.Rhs {
			inspectExpr(pass, rhs, state)
		}
		if len(x.Lhs) == len(x.Rhs) {
			for i, lhs := range x.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					assignString(pass, state, x.Rhs[i], ident)
				}
			}
		}
	case *ast.DeclStmt:
		if gen, ok := x.Decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, expr := range value.Values {
					inspectExpr(pass, expr, state)
				}
				if len(value.Names) == len(value.Values) {
					for i, name := range value.Names {
						assignString(pass, state, value.Values[i], name)
					}
				}
			}
		}
	case *ast.IfStmt:
		if x.Init != nil {
			state = processStmt(pass, x.Init, state)
		}
		inspectExpr(pass, x.Cond, state)
		thenState := processBlock(pass, x.Body, cloneStringState(state))
		elseState := cloneStringState(state)
		if x.Else != nil {
			elseState = processStmt(pass, x.Else, elseState)
		}
		return mergeStringStates(thenState, elseState)
	case *ast.ForStmt:
		base := cloneStringState(state)
		if x.Init != nil {
			base = processStmt(pass, x.Init, base)
		}
		if x.Cond != nil {
			inspectExpr(pass, x.Cond, base)
		}
		body := processBlock(pass, x.Body, cloneStringState(base))
		if x.Post != nil {
			body = processStmt(pass, x.Post, body)
		}
		return mergeStringStates(base, body)
	case *ast.RangeStmt:
		inspectExpr(pass, x.X, state)
		body := processBlock(pass, x.Body, cloneStringState(state))
		return mergeStringStates(state, body)
	case *ast.SwitchStmt:
		if x.Init != nil {
			state = processStmt(pass, x.Init, state)
		}
		if x.Tag != nil {
			inspectExpr(pass, x.Tag, state)
		}
		branches := []stringState{cloneStringState(state)}
		for _, raw := range x.Body.List {
			clause := raw.(*ast.CaseClause)
			branch := cloneStringState(state)
			for _, expr := range clause.List {
				inspectExpr(pass, expr, branch)
			}
			branches = append(branches, processBlock(pass, &ast.BlockStmt{List: clause.Body}, branch))
		}
		return mergeStringStates(branches...)
	case *ast.TypeSwitchStmt:
		state = processStmt(pass, x.Assign, state)
		branches := []stringState{cloneStringState(state)}
		for _, raw := range x.Body.List {
			clause := raw.(*ast.CaseClause)
			branches = append(branches, processBlock(pass, &ast.BlockStmt{List: clause.Body}, cloneStringState(state)))
		}
		return mergeStringStates(branches...)
	case *ast.SelectStmt:
		branches := []stringState{cloneStringState(state)}
		for _, raw := range x.Body.List {
			clause := raw.(*ast.CommClause)
			branch := cloneStringState(state)
			if clause.Comm != nil {
				branch = processStmt(pass, clause.Comm, branch)
			}
			branches = append(branches, processBlock(pass, &ast.BlockStmt{List: clause.Body}, branch))
		}
		return mergeStringStates(branches...)
	default:
		ast.Inspect(stmt, func(n ast.Node) bool {
			if expr, ok := n.(ast.Expr); ok {
				inspectExpr(pass, expr, state)
				return false
			}
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			return true
		})
	}
	return state
}

func processBlock(pass *lint.Pass, block *ast.BlockStmt, state stringState) stringState {
	for _, stmt := range block.List {
		state = processStmt(pass, stmt, state)
	}
	return state
}

// Check reports panic(error) calls and unwrapped error propagation in pass.
func (WrapPropagatedErrors) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				processBlock(pass, fn.Body, make(stringState))
			}
		}
	}
	return nil
}
