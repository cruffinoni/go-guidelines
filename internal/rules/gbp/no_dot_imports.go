package gbp

import (
	"go/ast"
	"go/token"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// NoDotImports flags dot imports, which hide where identifiers come from.
type NoDotImports struct{}

// Meta returns the GBP001 identity, severity, and documentation.
func (NoDotImports) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP001",
		Name:    "no-dot-imports",
		Summary: "Flags dot imports.",
		Description: "Reports every `import . \"pkg\"` spec, pointing at the dot. Dot imports pull a " +
			"package's exported identifiers into the local scope unqualified, so readers cannot tell " +
			"which package an identifier comes from. Use an explicit package name or alias instead.",
		Severity: lint.SeverityError,
	}
}

// Check reports every dot import in pass.
func (NoDotImports) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, declaration := range file.Decls {
			imports, ok := declaration.(*ast.GenDecl)
			if !ok || imports.Tok != token.IMPORT {
				continue
			}
			for _, spec := range imports.Specs {
				importSpec, ok := spec.(*ast.ImportSpec)
				if !ok || importSpec.Name == nil || importSpec.Name.Name != "." {
					continue
				}
				pass.Reportf(importSpec.Name.Pos(), "avoid dot imports; use an explicit package name")
			}
		}
	}
	return nil
}
