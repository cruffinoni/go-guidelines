package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// NoInit flags package-level init functions.
type NoInit struct{}

// Meta returns the GBP016 identity, severity, and documentation.
func (NoInit) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP016",
		Name:    "no-init",
		Summary: "Flags package-level init functions.",
		Description: "Reports every `func init()` declaration. Hidden execution before `main` " +
			"complicates testing and reasoning about program startup; prefer an explicit constructor or " +
			"setup function that callers invoke themselves.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports every init function declaration in pass.
func (NoInit) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil && f.Name.Name == "init" {
				pass.Reportf(f.Pos(), "avoid init; prefer explicit setup")
			}
		}
	}
	return nil
}
