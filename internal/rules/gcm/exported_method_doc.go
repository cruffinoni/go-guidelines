package gcm

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ExportedMethodDoc flags exported methods without a documentation comment.
type ExportedMethodDoc struct{}

// Meta returns the GCM003 identity, severity, and documentation.
func (ExportedMethodDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM003",
		Name:    "exported-method-doc",
		Summary: "Flags exported methods without a documentation comment.",
		Description: "Reports an exported method on an exported type that has no doc comment. Methods " +
			"carry the same documentation expectations as functions. Methods on unexported types are " +
			"skipped: `go doc` does not show them.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported methods in pass that have no documentation comment.
func (ExportedMethodDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for f := range exportedFuncs(pass, file) {
			if f.Recv == nil || f.Doc != nil || !ast.IsExported(receiverTypeName(f.Recv.List[0].Type)) {
				continue
			}
			pass.Report(lint.Diagnostic{Pos: f.Pos(), Message: "exported function or method is missing a documentation comment"})
		}
	}
	return nil
}

// receiverTypeName returns the base type name of a method receiver type such
// as T, *T, or *G[K, V], or "" if expr has no such name.
func receiverTypeName(expr ast.Expr) string {
	for {
		switch e := expr.(type) {
		case *ast.Ident:
			return e.Name
		case *ast.StarExpr:
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		default:
			return ""
		}
	}
}
