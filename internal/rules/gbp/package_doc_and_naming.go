package gbp

import "github.com/cruffinoni/go-guidelines/internal/lint"

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
			"reports a package with no package-level doc comment. Vague names grow into unrelated " +
			"catch-alls, and the package comment is the first thing readers see in `go doc`. Name the " +
			"package after its domain and add a package comment.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports vague package names and missing package docs in pass.
func (PackageDocAndNaming) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		switch file.Name.Name {
		case "util", "utils", "common", "helpers":
			pass.Reportf(file.Name.Pos(), "avoid vague package names; use a domain-specific name")
		}
		if file.Doc == nil {
			pass.Reportf(file.Name.Pos(), "package is missing package-level documentation")
		}
	}
	return nil
}
