package gbp

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// MethodCandidateForPointerReceiver flags functions that take one local
// struct as their only parameter.
type MethodCandidateForPointerReceiver struct{}

// Meta returns the GBP018 identity, severity, and documentation.
func (MethodCandidateForPointerReceiver) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP018",
		Name:    "method-candidate-for-pointer-receiver",
		Summary: "Flags functions that take a local struct as their only parameter.",
		Description: "Reports a top-level function whose only parameter is a value or pointer to a " +
			"defined struct type in the same package. A function that exists only to operate on one " +
			"type's data is more discoverable, and keeps related logic co-located, as a method on that " +
			"type. Constructors and functions with additional parameters are left to review; GBP027 " +
			"complements this rule for helpers with additional parameters or indirect call paths.",
		Severity:   lint.SeverityInfo,
		NeedsTypes: true,
	}
}

// isConstructorName reports whether name follows the common constructor
// naming convention.
func isConstructorName(name string) bool {
	return strings.HasPrefix(name, "New") || strings.HasPrefix(name, "new")
}

// localStructType returns the named local struct type behind t, if any.
func localStructType(pass *lint.Pass, t types.Type) *types.Named {
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
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil
	}
	return named
}

// isSingleStructParameter reports whether f has exactly one parameter whose
// type is a local struct, the shape GBP018 reports.
func isSingleStructParameter(pass *lint.Pass, f *ast.FuncDecl) bool {
	if parameterCount(f.Type.Params) != 1 {
		return false
	}
	return localStructType(pass, pass.TypesInfo.TypeOf(f.Type.Params.List[0].Type)) != nil
}

// Check reports functions that could be methods in pass.
func (MethodCandidateForPointerReceiver) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || f.Recv != nil || isConstructorName(f.Name.Name) || !isSingleStructParameter(pass, f) {
				continue
			}
			pass.Reportf(f.Pos(), "function taking a struct as its only parameter may belong as a method")
		}
	}
	return nil
}
