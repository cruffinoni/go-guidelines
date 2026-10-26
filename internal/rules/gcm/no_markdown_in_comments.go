package gcm

import (
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// NoMarkdownInComments flags Markdown-style formatting inside Go comments.
type NoMarkdownInComments struct{}

// Meta returns the GCM006 identity, severity, and documentation.
func (NoMarkdownInComments) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM006",
		Name:    "no-markdown-in-comments",
		Summary: "Flags Markdown-style formatting inside Go comments.",
		Description: "Reports a comment line containing `**`, a Markdown link (`](`), or a multi-hash " +
			"heading (`// ##`). Go doc comments support only their own small syntax; Markdown emphasis, " +
			"links, and multi-level headings render literally in `go doc` output. A single `#` heading is " +
			"allowed and not flagged.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports comment lines in pass containing Markdown emphasis, links, or
// multi-hash headings.
func (NoMarkdownInComments) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, group := range file.Comments {
			for _, c := range group.List {
				line := strings.TrimSpace(c.Text)
				if strings.Contains(line, "**") || strings.Contains(line, "](") || strings.HasPrefix(line, "// ##") {
					pass.Report(lint.Diagnostic{Pos: c.Pos(), Message: "avoid Markdown formatting in Go comments"})
				}
			}
		}
	}
	return nil
}
