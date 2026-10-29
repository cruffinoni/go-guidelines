package gcm

import "github.com/cruffinoni/go-guidelines/internal/lint"

// NoTautologicalDoc flags documentation comments that merely restate the
// function name.
type NoTautologicalDoc struct{}

// Meta returns the GCM009 identity, severity, and documentation.
func (NoTautologicalDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM009",
		Name:    "no-tautological-doc",
		Summary: "Flags documentation comments that merely restate the function name.",
		Description: "Reports an exported function or method whose doc comment reduces to `<Name> " +
			"returns <words of name>`, or for `GetX` names to `<name> returns <x>`, ignoring case and a " +
			"trailing period. A comment that only restates the identifier adds nothing beyond what the " +
			"signature already says.",
		Severity: lint.SeverityInfo,
	}
}

// Check reports exported functions and methods in pass whose documentation
// comment is tautological.
func (NoTautologicalDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for f := range exportedFuncs(pass, file) {
			if f.Doc == nil {
				continue
			}
			if tautologicalDoc(f.Name.Name, f.Doc.Text()) {
				pass.Report(lint.Diagnostic{Pos: f.Pos(), Message: "documentation comment appears tautological and low-value"})
			}
		}
	}
	return nil
}
