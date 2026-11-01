package gcm

import (
	"go/ast"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ExportedTypeDoc flags exported types without a documentation comment.
type ExportedTypeDoc struct{}

// Meta returns the GCM001 identity, severity, and documentation.
func (ExportedTypeDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM001",
		Name:    "exported-type-doc",
		Summary: "Flags exported types without a documentation comment.",
		Description: "Reports an exported type declaration with neither its own doc comment nor one on " +
			"its enclosing declaration group. Types whose name ends in `Config` are left to GCM004. Every " +
			"exported name should carry a doc comment so it reads correctly in `go doc`.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported, non-Config type declarations in pass that have no
// documentation comment of their own or on their declaration group.
func (ExportedTypeDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range g.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ast.IsExported(ts.Name.Name) || strings.HasSuffix(ts.Name.Name, "Config") {
					continue
				}
				if ts.Doc == nil && g.Doc == nil {
					pass.Report(lint.Diagnostic{Pos: ts.Pos(), Message: "exported type is missing a documentation comment"})
				}
			}
		}
	}
	return nil
}
