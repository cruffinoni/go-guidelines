// Package gcm implements the Go comment (GCM) rules.
package gcm

import "github.com/cruffinoni/go-guidelines/internal/lint"

// Rules returns every GCM rule, in stable ID order.
func Rules() []lint.Rule {
	return []lint.Rule{
		ExportedTypeDoc{},
		ExportedFunctionDoc{},
		ExportedMethodDoc{},
		ConfigTypeDoc{},
		ExportedErrorVarDoc{},
		NoMarkdownInComments{},
		CommentLineLength{},
		ExampleOutputComment{},
		NoTautologicalDoc{},
	}
}
