package gcm

import (
	"go/token"
	"strings"
	"unicode/utf8"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// maxCommentLineLength is the longest comment line CommentLineLength accepts,
// in Unicode code points of the visible comment text.
const maxCommentLineLength = 120

// CommentLineLength flags comment lines longer than 120 characters.
type CommentLineLength struct{}

// Meta returns the GCM007 identity, severity, and documentation.
func (CommentLineLength) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM007",
		Name:    "comment-line-length",
		Summary: "Flags comment lines longer than 120 characters.",
		Description: "Reports any physical comment line longer than 120 characters, measured in " +
			"Unicode code points after removing the comment markers. Long unwrapped comment lines are " +
			"harder to read in a terminal or a side-by-side diff; wrap around the project's line length.",
		Severity: lint.SeverityWarning,
	}
}

// commentLineContent returns the visible text of one physical comment line:
// comment delimiters are removed, and a leading '*' on an interior block
// comment line is ignored.
func commentLineContent(line string, first, last bool) string {
	line = strings.TrimSuffix(line, "\r")
	if first {
		switch {
		case strings.HasPrefix(line, "//"):
			line = strings.TrimPrefix(line, "//")
		case strings.HasPrefix(line, "/*"):
			line = strings.TrimPrefix(line, "/*")
		}
	}
	if last {
		line = strings.TrimSuffix(line, "*/")
	}
	if !first && !last {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "*") {
			line = strings.TrimPrefix(trimmed, "*")
		}
	}
	return line
}

// Check reports comment lines in pass longer than maxCommentLineLength.
func (CommentLineLength) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for _, group := range file.Comments {
			for _, c := range group.List {
				lines := strings.Split(c.Text, "\n")
				offset := 0
				for i, line := range lines {
					content := commentLineContent(line, i == 0, i == len(lines)-1)
					if utf8.RuneCountInString(content) > maxCommentLineLength {
						pass.Report(lint.Diagnostic{
							Pos:     c.Pos() + token.Pos(offset),
							Message: "comment line exceeds 120 characters",
						})
					}
					offset += len(line) + 1
				}
			}
		}
	}
	return nil
}
