package gcm

import "github.com/cruffinoni/go-guidelines/internal/lint"

// maxCommentLineLength is the longest comment line CommentLineLength accepts,
// in bytes of the verbatim parsed text.
const maxCommentLineLength = 120

// CommentLineLength flags comment lines longer than 120 characters.
type CommentLineLength struct{}

// Meta returns the GCM007 identity, severity, and documentation.
func (CommentLineLength) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM007",
		Name:    "comment-line-length",
		Summary: "Flags comment lines longer than 120 characters.",
		Description: "Reports any comment line, as recorded verbatim by the parser, longer than 120 " +
			"characters. Long unwrapped comment lines are harder to read in a terminal or a side-by-side " +
			"diff; wrap around the project's line length.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports comment lines in pass longer than maxCommentLineLength.
func (CommentLineLength) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, group := range file.Comments {
			for _, c := range group.List {
				if len(c.Text) > maxCommentLineLength {
					pass.Report(lint.Diagnostic{Pos: c.Pos(), Message: "comment line exceeds 120 characters"})
				}
			}
		}
	}
	return nil
}
