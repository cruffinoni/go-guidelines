package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// PackageDocAndNaming flags vague package names and missing package
// documentation.
type PackageDocAndNaming struct{}

// Meta returns the GBP002 identity, severity, and documentation.
func (PackageDocAndNaming) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP002",
		Name:    "package-doc-and-naming",
		Summary: "Flags vague package names and missing package documentation.",
		Description: "Reports a package named `util`, `utils`, `common`, or `helpers`, and separately " +
			"reports a package with no package-level doc comment in any of its non-test files. Each " +
			"is reported once per package. Vague names grow into unrelated catch-alls, and the package " +
			"comment is the first thing readers see in `go doc`. Name the package after its domain and " +
			"add a package comment.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports vague package names and missing package docs in pass, once
// per package, at the package clause of its first non-test, non-generated
// file. Test files are ignored, so an external test package reports
// nothing; a generated file's package doc still documents the package.
func (PackageDocAndNaming) Check(pass *lint.Pass) error {
	var first *ast.File
	documented := false
	for _, file := range pass.Files {
		if pass.IsTestFile(file) {
			continue
		}
		if first == nil && !ast.IsGenerated(file) {
			first = file
		}
		documented = documented || file.Doc != nil
	}
	if first == nil {
		return nil
	}
	switch first.Name.Name {
	case "util", "utils", "common", "helpers":
		pass.Reportf(first.Name.Pos(), "avoid vague package names; use a domain-specific name")
	}
	if !documented {
		pass.Reportf(first.Name.Pos(), "package is missing package-level documentation")
	}
	return nil
}
