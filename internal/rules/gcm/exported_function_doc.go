package gcm

import "github.com/cruffinoni/go-guidelines/internal/lint"

// ExportedFunctionDoc flags exported top-level functions without a
// documentation comment.
type ExportedFunctionDoc struct{}

// Meta returns the GCM002 identity, severity, and documentation.
func (ExportedFunctionDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM002",
		Name:    "exported-function-doc",
		Summary: "Flags exported top-level functions without a documentation comment.",
		Description: "Reports an exported function with no receiver and no doc comment, except " +
			"`TestXxx` functions in test files. Undocumented exported functions show up blank in `go doc` " +
			"and pkgsite.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported, receiver-less functions in pass that have no
// documentation comment.
func (ExportedFunctionDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for f := range exportedFuncs(pass, file) {
			if f.Recv != nil || f.Doc != nil {
				continue
			}
			pass.Report(lint.Diagnostic{Pos: f.Pos(), Message: "exported function or method is missing a documentation comment"})
		}
	}
	return nil
}
