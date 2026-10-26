package gbp

import (
	"go/ast"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// DocumentSliceJSONNilVsEmpty flags slice-typed JSON fields that don't
// document nil-versus-empty encoding.
type DocumentSliceJSONNilVsEmpty struct{}

// Meta returns the GBP010 identity, severity, and documentation.
func (DocumentSliceJSONNilVsEmpty) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP010",
		Name:    "document-slice-json-nil-vs-empty",
		Summary: "Flags slice-typed JSON fields that don't document nil-versus-empty encoding.",
		Description: "Reports a struct field of slice type with a `json` tag that lacks `omitempty` and " +
			"is not `json:\"-\"`. Without `omitempty`, a nil slice encodes as `null` while an empty one " +
			"encodes as `[]`, a distinction callers may not expect; add `omitempty` or otherwise document " +
			"the encoding.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports undocumented slice JSON fields in pass.
func (DocumentSliceJSONNilVsEmpty) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.Field)
			if !ok || f.Tag == nil {
				return true
			}
			tag := strings.Trim(f.Tag.Value, "`")
			if !strings.Contains(tag, "json:") {
				return true
			}
			if _, ok := f.Type.(*ast.ArrayType); !ok {
				return true
			}
			if strings.Contains(tag, "omitempty") || strings.Contains(tag, "json:\"-\"") {
				return true
			}
			pass.Reportf(f.Pos(), "slice JSON field should document nil-versus-empty encoding (consider omitempty)")
			return true
		})
	}
	return nil
}
