package rules

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// GBP001 detects dot imports, which make the origin of imported identifiers
// unclear. It emits one finding at the dot for every dot import in a file.
type GBP001 struct{}

func (GBP001) ID() string { return "GBP001" }

func (GBP001) Analyze(filename string, source []byte) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, source, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse Go source %q: %w", filename, err)
	}

	findings := make([]Finding, 0)
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
			position := fset.Position(importSpec.Name.Pos())
			findings = append(findings, Finding{
				RuleID:   "GBP001",
				Severity: SeverityError,
				Message:  "avoid dot imports; use an explicit package name",
				File:     filename,
				Line:     position.Line,
				Column:   position.Column,
			})
		}
	}
	return findings, nil
}
