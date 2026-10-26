package gbp

import (
	"go/ast"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// StructuredLogging flags unstructured log package calls.
type StructuredLogging struct{}

// Meta returns the GBP011 identity, severity, and documentation.
func (StructuredLogging) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP011",
		Name:    "structured-logging",
		Summary: "Flags unstructured log package calls.",
		Description: "Reports calls whose name starts with `log.Print`, `log.Fatal`, or `log.Panic`. " +
			"These calls produce plain-text lines with no key-value structure, which is harder to query " +
			"and correlate than structured logs. Use `log/slog` with key-value pairs instead.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports calls to log package print functions in pass.
func (StructuredLogging) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				name := callName(c.Fun)
				if strings.HasPrefix(name, "log.Print") || strings.HasPrefix(name, "log.Fatal") || strings.HasPrefix(name, "log.Panic") {
					pass.Reportf(c.Pos(), "prefer structured logging over log package print calls")
				}
			}
			return true
		})
	}
	return nil
}
