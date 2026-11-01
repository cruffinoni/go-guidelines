package gbp

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ConsistentReceivers flags defined types that mix value and pointer
// receiver methods.
type ConsistentReceivers struct{}

// Meta returns the GBP022 identity, severity, and documentation.
func (ConsistentReceivers) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP022",
		Name:    "consistent-receivers",
		Summary: "Flags types that mix value and pointer receiver methods.",
		Description: "Reports a defined type that has both value-receiver and pointer-receiver methods. " +
			"Mixing receiver forms makes it unclear whether a method can safely be called through an " +
			"interface or on a copied value; use one receiver form consistently.",
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// receiverMethod records one method declaration for receiver consistency.
type receiverMethod struct {
	pos     token.Pos
	pointer bool
	inScope bool
}

// receiverGroup collects methods declared on one local defined type.
type receiverGroup struct {
	name    string
	methods []receiverMethod
}

// receiverTypeObject returns the local defined type object behind a method
// receiver and whether the receiver is a pointer.
func receiverTypeObject(pass *lint.Pass, expr ast.Expr) (types.Object, bool) {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return nil, false
	}
	pointer := false
	if ptr, ok := t.(*types.Pointer); ok {
		pointer = true
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return nil, false
	}
	return named.Obj(), pointer
}

// Check reports defined types that mix value and pointer receivers in pass.
func (ConsistentReceivers) Check(pass *lint.Pass) error {
	groups := make(map[types.Object]*receiverGroup)
	for _, file := range pass.Files {
		inScope := pass.FileInScope(file)
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || f.Recv == nil || len(f.Recv.List) == 0 {
				continue
			}
			obj, pointer := receiverTypeObject(pass, f.Recv.List[0].Type)
			if obj == nil || obj.Pkg() != pass.Pkg {
				continue
			}
			group := groups[obj]
			if group == nil {
				group = &receiverGroup{name: obj.Name()}
				groups[obj] = group
			}
			group.methods = append(group.methods, receiverMethod{
				pos:     f.Pos(),
				pointer: pointer,
				inScope: inScope,
			})
		}
	}

	for _, group := range groups {
		var hasValue, hasPointer bool
		for _, method := range group.methods {
			if method.pointer {
				hasPointer = true
			} else {
				hasValue = true
			}
		}
		if !hasValue || !hasPointer {
			continue
		}
		slices.SortFunc(group.methods, func(a, b receiverMethod) int {
			return int(a.pos - b.pos)
		})
		for _, method := range group.methods {
			if method.inScope {
				pass.Reportf(method.pos, "type %s has both value and pointer receiver methods; use one receiver form consistently", group.name)
				break
			}
		}
	}
	return nil
}
