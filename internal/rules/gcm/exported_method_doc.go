package gcm

import "github.com/cruffinoni/go-guidelines/internal/lint"

// ExportedMethodDoc flags exported methods without a documentation comment.
type ExportedMethodDoc struct{}

// Meta returns the GCM003 identity, severity, and documentation.
func (ExportedMethodDoc) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM003",
		Name:    "exported-method-doc",
		Summary: "Flags exported methods without a documentation comment.",
		Description: "Reports an exported method, one with a non-nil receiver, that has no doc comment. " +
			"Methods carry the same documentation expectations as functions.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported methods in pass that have no documentation comment.
func (ExportedMethodDoc) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for f := range exportedFuncs(pass, file) {
			if f.Recv == nil || f.Doc != nil {
				continue
			}
			pass.Report(lint.Diagnostic{Pos: f.Pos(), Message: "exported function or method is missing a documentation comment"})
		}
	}
	return nil
}
