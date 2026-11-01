package gcm

import (
	"go/ast"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ConfigTypeDoc flags exported *Config types without a documentation comment.
type ConfigTypeDoc struct{}

// Meta returns the GCM004 identity, severity, and documentation.
func (ConfigTypeDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM004",
		Name:    "config-type-doc",
		Summary: "Flags exported *Config types without a documentation comment.",
		Description: "Reports an exported type whose name ends in `Config` that has neither its own doc " +
			"comment nor one on its enclosing declaration group. Configuration types should explain what " +
			"they configure and how values are defaulted.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported *Config type declarations in pass that have no
// documentation comment of their own or on their declaration group.
func (ConfigTypeDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range g.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ast.IsExported(ts.Name.Name) || !strings.HasSuffix(ts.Name.Name, "Config") {
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
