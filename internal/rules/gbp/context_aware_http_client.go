package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ContextAwareHTTPClient flags HTTP calls that skip context and connection
// reuse.
type ContextAwareHTTPClient struct{}

// Meta returns the GBP006 identity, severity, and documentation.
func (ContextAwareHTTPClient) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP006",
		Name:    "context-aware-http-client",
		Summary: "Flags HTTP calls that skip context and connection reuse.",
		Description: "Reports calls to `http.Get` and to `http.NewRequest`. `http.Get` builds a request " +
			"with no context and no way to cancel it; `http.NewRequest` should become " +
			"`http.NewRequestWithContext` once a context is available. Build requests with a context and " +
			"reuse a shared `http.Client`.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports http.Get and http.NewRequest calls in pass.
func (ContextAwareHTTPClient) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				switch callName(c.Fun) {
				case "http.Get":
					pass.Reportf(c.Pos(), "avoid http.Get; use a context-aware request and reusable client")
				case "http.NewRequest":
					pass.Reportf(c.Pos(), "use http.NewRequestWithContext when request cancellation is available")
				}
			}
			return true
		})
	}
	return nil
}
