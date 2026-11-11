package gbp

import (
	"go/ast"
	"go/types"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// TypeOwnedHelper flags unexported helpers whose call paths and data usage
// show they are owned by a single local struct's methods.
type TypeOwnedHelper struct{}

// Meta returns the GBP027 identity, severity, and documentation.
func (TypeOwnedHelper) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP027",
		Name:    "type-owned-helper",
		Summary: "Flags unexported helpers owned by a single local struct type's methods.",
		Description: "Reports an unexported, non-generic package-level helper whose known production " +
			"call paths all originate from methods of one local struct type and whose signature or " +
			"call sites show that it consumes that type's state. Exclusive use alone is not enough: an " +
			"independent transformation stays a function even with a single caller. References from " +
			"tests do not affect ownership, and helpers used as function values, called from " +
			"package-level code, reached through more than one type, or supported only by a naming " +
			"convention are left alone. Advisory: prefer a method only when the behavior is genuinely " +
			"owned by the type. Complements GBP018, which covers helpers whose only parameter is a " +
			"local struct.",
		Severity:   lint.SeverityInfo,
		NeedsTypes: true,
	}
}

// helperRefKind classifies one reference to a package-level helper.
type helperRefKind uint8

const (
	// refMethod is a call made from a method of a local struct type.
	refMethod helperRefKind = iota + 1
	// refCaller is a call made from another eligible package-level helper.
	refCaller
	// refEscape is a non-call use such as a function value or an argument.
	refEscape
	// refPackage is a use from package-level code or from a caller whose
	// ownership cannot be tracked.
	refPackage
	// refGenerated is a use from generated code.
	refGenerated
)

// helperRef records one reference to a package-level helper.
type helperRef struct {
	kind   helperRefKind
	owner  *types.Named  // refMethod: local struct whose method contains the ref
	caller *types.Func   // refCaller: eligible helper that contains the ref
	call   *ast.CallExpr // direct call site for method and caller refs
	from   *ast.FuncDecl // enclosing declaration, for ownership evidence
}

// helperState is one eligible package-level helper declaration.
type helperState struct {
	decl      *ast.FuncDecl
	candidate bool
	refs      []helperRef
}

// typeOwnedAnalysis holds the per-package GBP027 reference graph.
type typeOwnedAnalysis struct {
	pass    *lint.Pass
	helpers map[types.Object]*helperState
	order   []*helperState
}

// packageFunc returns the package-level function object that id names, if
// any. Methods are not part of the helper graph and are filtered by map
// lookups.
func packageFunc(pass *lint.Pass, id *ast.Ident) *types.Func {
	obj, ok := pass.TypesInfo.Uses[id].(*types.Func)
	if !ok || obj.Pkg() != pass.Pkg {
		return nil
	}
	return obj
}

// funcDeclObject returns f's function object when it belongs to the package.
func funcDeclObject(pass *lint.Pass, f *ast.FuncDecl) *types.Func {
	obj, ok := pass.TypesInfo.Defs[f.Name].(*types.Func)
	if !ok || obj.Pkg() != pass.Pkg {
		return nil
	}
	return obj
}

// isHelperDecl reports whether f may participate in the ownership graph.
// Generated and test files are excluded by the caller.
func isHelperDecl(f *ast.FuncDecl, generated bool) bool {
	if generated || f.Recv != nil || f.Body == nil || f.Name.IsExported() || f.Name.Name == "init" {
		return false
	}
	return f.Type.TypeParams == nil || len(f.Type.TypeParams.List) == 0
}

// isReportableHelper reports whether f is a helper GBP027 may report.
func isReportableHelper(pass *lint.Pass, f *ast.FuncDecl) bool {
	if isConstructorName(f.Name.Name) || parameterCount(f.Type.Params) == 0 {
		return false
	}
	return !isSingleStructParameter(pass, f)
}

// originStructType returns the local struct type behind t, normalizing
// generic instantiations to their origin type.
func originStructType(pass *lint.Pass, t types.Type) *types.Named {
	named := localStructType(pass, t)
	if named == nil {
		return nil
	}
	return named.Origin()
}

// localNamedType returns the local named type behind t, if any, without
// requiring a struct underlying type.
func localNamedType(pass *lint.Pass, t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() != pass.Pkg {
		return nil
	}
	return named
}

// calleeIdent returns the base identifier of a call target, unwrapping
// parentheses and generic instantiation brackets.
func calleeIdent(expr ast.Expr) *ast.Ident {
	switch x := expr.(type) {
	case *ast.Ident:
		return x
	case *ast.ParenExpr:
		return calleeIdent(x.X)
	case *ast.IndexExpr:
		return calleeIdent(x.X)
	case *ast.IndexListExpr:
		return calleeIdent(x.X)
	}
	return nil
}

// selectorBase returns the identifier a selector expression is rooted in,
// unwrapping parentheses and pointer dereferences.
func selectorBase(expr ast.Expr) *ast.Ident {
	switch x := expr.(type) {
	case *ast.Ident:
		return x
	case *ast.ParenExpr:
		return selectorBase(x.X)
	case *ast.StarExpr:
		return selectorBase(x.X)
	}
	return nil
}

// receiverObject returns the named receiver object of method f, if any.
func receiverObject(pass *lint.Pass, f *ast.FuncDecl) types.Object {
	if f.Recv == nil || len(f.Recv.List) == 0 || len(f.Recv.List[0].Names) == 0 {
		return nil
	}
	return pass.TypesInfo.Defs[f.Recv.List[0].Names[0]]
}

// collectDeclarations indexes every eligible package-level helper.
func (a *typeOwnedAnalysis) collectDeclarations() {
	for _, file := range a.pass.Files {
		if a.pass.IsTestFile(file) {
			continue
		}
		generated := ast.IsGenerated(file)
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || !isHelperDecl(f, generated) {
				continue
			}
			obj := funcDeclObject(a.pass, f)
			if obj == nil {
				continue
			}
			state := &helperState{decl: f, candidate: isReportableHelper(a.pass, f)}
			a.helpers[obj] = state
			a.order = append(a.order, state)
		}
	}
}

// collectPackageRefs records references from package-level declarations.
func (a *typeOwnedAnalysis) collectPackageRefs(decl *ast.GenDecl, generated bool) {
	for _, spec := range decl.Specs {
		ast.Inspect(spec, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := packageFunc(a.pass, id)
			if obj == nil {
				return true
			}
			state := a.helpers[obj]
			if state == nil {
				return true
			}
			kind := refPackage
			if generated {
				kind = refGenerated
			}
			state.refs = append(state.refs, helperRef{kind: kind})
			return true
		})
	}
}

// collectFuncRefs records references from the body of f.
func (a *typeOwnedAnalysis) collectFuncRefs(f *ast.FuncDecl, generated bool) {
	obj := funcDeclObject(a.pass, f)
	kind := refPackage
	var owner *types.Named
	var caller *types.Func
	switch {
	case generated:
		kind = refGenerated
	case f.Recv != nil:
		if len(f.Recv.List) > 0 {
			owner = originStructType(a.pass, a.pass.TypesInfo.TypeOf(f.Recv.List[0].Type))
		}
		if owner != nil {
			kind = refMethod
		}
	case obj != nil && a.helpers[obj] != nil:
		kind = refCaller
		caller = obj
	}

	callees := make(map[*ast.Ident]*ast.CallExpr)
	ast.Inspect(f.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id := calleeIdent(call.Fun); id != nil {
			callees[id] = call
		}
		return true
	})
	ast.Inspect(f.Body, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		obj := packageFunc(a.pass, id)
		if obj == nil {
			return true
		}
		state := a.helpers[obj]
		if state == nil {
			return true
		}
		ref := helperRef{kind: kind, owner: owner, caller: caller, from: f}
		if kind == refMethod || kind == refCaller {
			call, called := callees[id]
			if !called {
				ref.kind = refEscape
			} else {
				ref.call = call
			}
		}
		state.refs = append(state.refs, ref)
		return true
	})
}

// collectReferences walks every production declaration and records how each
// helper is used. Test files are skipped entirely: they never anchor or
// disqualify ownership.
func (a *typeOwnedAnalysis) collectReferences() {
	for _, file := range a.pass.Files {
		if a.pass.IsTestFile(file) {
			continue
		}
		generated := ast.IsGenerated(file)
		for _, decl := range file.Decls {
			switch x := decl.(type) {
			case *ast.GenDecl:
				a.collectPackageRefs(x, generated)
			case *ast.FuncDecl:
				if x.Body != nil {
					a.collectFuncRefs(x, generated)
				}
			}
		}
	}
}

// methodOwners returns every local struct type whose methods reference a
// helper.
func (a *typeOwnedAnalysis) methodOwners() []*types.Named {
	var owners []*types.Named
	seen := make(map[*types.Named]bool)
	for _, state := range a.order {
		for _, ref := range state.refs {
			if ref.kind != refMethod || seen[ref.owner] {
				continue
			}
			seen[ref.owner] = true
			owners = append(owners, ref.owner)
		}
	}
	return owners
}

// hasMethodOwner reports whether a method of owner references state.
func (a *typeOwnedAnalysis) hasMethodOwner(state *helperState, owner *types.Named) bool {
	for _, ref := range state.refs {
		if ref.kind == refMethod && ref.owner == owner {
			return true
		}
	}
	return false
}

// calledFromSet reports whether a helper in set references state.
func (a *typeOwnedAnalysis) calledFromSet(state *helperState, set map[*helperState]bool) bool {
	for _, ref := range state.refs {
		if ref.kind != refCaller {
			continue
		}
		if caller := a.helpers[ref.caller]; caller != nil && set[caller] {
			return true
		}
	}
	return false
}

// consistentWith reports whether every reference to state supports owner as
// its only owner, given the helpers already attributed to owner.
func (a *typeOwnedAnalysis) consistentWith(state *helperState, owner *types.Named, set map[*helperState]bool) bool {
	for _, ref := range state.refs {
		switch ref.kind {
		case refEscape, refPackage, refGenerated:
			return false
		case refMethod:
			if ref.owner != owner {
				return false
			}
		case refCaller:
			caller := a.helpers[ref.caller]
			if caller == nil || !set[caller] {
				return false
			}
		}
	}
	return true
}

// ownedClosure returns the helpers whose known call paths reach them only
// from methods of owner. The expansion admits helpers with a caller in the
// set; the pruning then removes any helper with a conflicting reference until
// the set is stable. Anchored recursion is therefore kept, while cycles with
// no method entry point never join the set.
func (a *typeOwnedAnalysis) ownedClosure(owner *types.Named) map[*helperState]bool {
	set := make(map[*helperState]bool)
	for _, state := range a.order {
		if a.hasMethodOwner(state, owner) {
			set[state] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for _, state := range a.order {
			if set[state] || !a.calledFromSet(state, set) {
				continue
			}
			set[state] = true
			changed = true
		}
	}
	for changed := true; changed; {
		changed = false
		for state := range set {
			if !a.consistentWith(state, owner, set) {
				delete(set, state)
				changed = true
			}
		}
	}
	return set
}

// resolveOwners attributes each helper to the single local struct type whose
// methods own it; conflicting attributions are suppressed.
func (a *typeOwnedAnalysis) resolveOwners() map[*helperState]*types.Named {
	resolved := make(map[*helperState]*types.Named)
	ambiguous := make(map[*helperState]bool)
	for _, owner := range a.methodOwners() {
		for state := range a.ownedClosure(owner) {
			if ambiguous[state] {
				continue
			}
			if previous, ok := resolved[state]; ok && previous != owner {
				delete(resolved, state)
				ambiguous[state] = true
				continue
			}
			resolved[state] = owner
		}
	}
	return resolved
}

// expressionRoot reports whether expr is rooted in one of roots, unwrapping
// the expression forms that derive a value from receiver state.
func expressionRoot(pass *lint.Pass, expr ast.Expr, roots map[types.Object]bool) bool {
	switch x := expr.(type) {
	case *ast.Ident:
		obj := pass.TypesInfo.Uses[x]
		return obj != nil && roots[obj]
	case *ast.ParenExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.UnaryExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.StarExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.SelectorExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.IndexExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.IndexListExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.SliceExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.TypeAssertExpr:
		return expressionRoot(pass, x.X, roots)
	case *ast.CallExpr:
		return expressionRoot(pass, x.Fun, roots)
	}
	return false
}

// ownerRoots returns the receiver and owner-typed parameters of the
// declaration containing ref.
func (a *typeOwnedAnalysis) ownerRoots(ref helperRef, owner *types.Named) map[types.Object]bool {
	roots := make(map[types.Object]bool)
	if ref.from == nil {
		return roots
	}
	if obj := receiverObject(a.pass, ref.from); obj != nil {
		roots[obj] = true
	}
	for _, field := range ref.from.Type.Params.List {
		if originStructType(a.pass, a.pass.TypesInfo.TypeOf(field.Type)) != owner {
			continue
		}
		for _, name := range field.Names {
			if obj := a.pass.TypesInfo.Defs[name]; obj != nil {
				roots[obj] = true
			}
		}
	}
	return roots
}

// isNeutralArgument reports whether expr carries no ownership information:
// it is a constant or nil.
func isNeutralArgument(pass *lint.Pass, expr ast.Expr) bool {
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok {
		return false
	}
	return tv.Value != nil || tv.IsNil()
}

// hasOwnerArgument reports whether a call site consistently passes owner data:
// every argument must be receiver state, an owner-typed parameter, or a
// constant, and at least one argument must be a value of a local defined type.
// Independent transformations over plain strings, slices, or numbers therefore
// stay functions.
func (a *typeOwnedAnalysis) hasOwnerArgument(ref helperRef, owner *types.Named) bool {
	if ref.call == nil {
		return false
	}
	roots := a.ownerRoots(ref, owner)
	if len(roots) == 0 {
		return false
	}
	owned := false
	for _, arg := range ref.call.Args {
		if expressionRoot(a.pass, arg, roots) {
			if localNamedType(a.pass, a.pass.TypesInfo.TypeOf(arg)) != nil {
				owned = true
			}
			continue
		}
		if !isNeutralArgument(a.pass, arg) {
			return false
		}
	}
	return owned
}

// readsOwnerState reports whether state has a parameter typed owner whose
// fields or methods the body accesses.
func (a *typeOwnedAnalysis) readsOwnerState(state *helperState, owner *types.Named) bool {
	params := make(map[types.Object]bool)
	for _, field := range state.decl.Type.Params.List {
		if originStructType(a.pass, a.pass.TypesInfo.TypeOf(field.Type)) != owner {
			continue
		}
		for _, name := range field.Names {
			if obj := a.pass.TypesInfo.Defs[name]; obj != nil {
				params[obj] = true
			}
		}
	}
	if len(params) == 0 {
		return false
	}
	found := false
	ast.Inspect(state.decl.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id := selectorBase(sel.X)
		if id == nil {
			return true
		}
		obj := a.pass.TypesInfo.Uses[id]
		if obj != nil && params[obj] {
			found = true
			return false
		}
		return true
	})
	return found
}

// passesOwnerData reports whether every known call site passes owner data.
func (a *typeOwnedAnalysis) passesOwnerData(state *helperState, owner *types.Named) bool {
	if len(state.refs) == 0 {
		return false
	}
	for _, ref := range state.refs {
		if !a.hasOwnerArgument(ref, owner) {
			return false
		}
	}
	return true
}

// evidenceMessage returns the finding message for state owned by owner, or ""
// when the inferred ownership is too weak to report.
func (a *typeOwnedAnalysis) evidenceMessage(state *helperState, owner *types.Named) string {
	name := owner.Obj().Name()
	if a.readsOwnerState(state, owner) {
		return "unexported helper is only reachable from methods of " + name +
			" and reads its state; consider declaring it as a method of " + name
	}
	if a.passesOwnerData(state, owner) {
		return "unexported helper is only reachable from methods of " + name +
			" and is always called with receiver data; consider declaring it as a method of " + name
	}
	return ""
}

// Check reports unexported helpers owned by a single local struct's methods.
func (TypeOwnedHelper) Check(pass *lint.Pass) error {
	if pass.Pkg == nil || len(pass.TypeErrors) > 0 {
		// Partial type information can hide references and fake exclusivity;
		// staying silent is the safe behavior for an advisory rule.
		return nil
	}
	analysis := &typeOwnedAnalysis{pass: pass, helpers: make(map[types.Object]*helperState)}
	analysis.collectDeclarations()
	analysis.collectReferences()
	owners := analysis.resolveOwners()
	for _, state := range analysis.order {
		if !state.candidate {
			continue
		}
		owner := owners[state]
		if owner == nil {
			continue
		}
		if message := analysis.evidenceMessage(state, owner); message != "" {
			pass.Reportf(state.decl.Name.Pos(), "%s", message)
		}
	}
	return nil
}
