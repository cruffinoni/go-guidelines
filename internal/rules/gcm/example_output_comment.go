package gcm

import (
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ExampleOutputComment flags Example functions without an Output comment.
type ExampleOutputComment struct{}

// Meta returns the GCM008 identity, severity, and documentation.
func (ExampleOutputComment) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GCM008",
		Name:    "example-output-comment",
		Summary: "Flags Example functions without an Output comment.",
		Description: "In test files, reports an exported function whose name starts with `Example` and " +
			"whose body has no `// Output:` or `// Unordered output:` comment. Without one of those " +
			"comments, `go test` never actually runs and checks the example.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports exported Example functions in pass whose body has no Output
// or Unordered output comment.
func (ExampleOutputComment) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		for f := range exportedFuncs(pass, file) {
			if !strings.HasPrefix(f.Name.Name, "Example") {
				continue
			}
			if !hasExampleOutput(file, f) {
				pass.Report(lint.Diagnostic{Pos: f.Pos(), Message: "example function is missing an Output comment"})
			}
		}
	}
	return nil
}
