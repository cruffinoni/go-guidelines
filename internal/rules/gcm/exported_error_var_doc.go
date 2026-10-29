package gcm

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ExportedErrorVarDoc flags exported Err-prefixed variables without a
// documentation comment.
type ExportedErrorVarDoc struct{}

// Meta returns the GCM005 identity, severity, and documentation.
func (ExportedErrorVarDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM005",
		Name:    "exported-error-var-doc",
		Summary: "Flags exported Err-prefixed variables without a documentation comment.",
		Description: "Reports an exported `var` whose name starts with `Err` and has neither its own " +
			"doc comment nor one on its enclosing declaration group. Error variables should document when " +
			"they are returned.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported Err-prefixed variables in pass that have no
// documentation comment of their own or on their declaration group.
func (ExportedErrorVarDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}
			for _, spec := range g.Specs {
				v, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range v.Names {
					if strings.HasPrefix(name.Name, "Err") && ast.IsExported(name.Name) && v.Doc == nil && g.Doc == nil {
						pass.Report(lint.Diagnostic{Pos: name.Pos(), Message: "exported error variable is missing a documentation comment"})
					}
				}
			}
		}
	}
	return nil
}
