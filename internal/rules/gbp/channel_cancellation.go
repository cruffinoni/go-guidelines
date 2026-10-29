package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ChannelCancellation flags channel-based functions with no visible
// cancellation path.
type ChannelCancellation struct{}

// Meta returns the GBP014 identity, severity, and documentation.
func (ChannelCancellation) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP014",
		Name:    "channel-cancellation",
		Summary: "Flags channel-based functions with no visible cancellation path.",
		Description: "Reports a function that uses a channel, as a parameter, local variable, or " +
			"send/receive, but has neither a `select` statement nor a reference to a context's `Done` " +
			"channel. A channel pipeline with no cancellation path can block forever; give every stage a " +
			"way to observe cancellation.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports channel-based functions lacking a cancellation path in pass.
func (ChannelCancellation) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			hasChan, hasSelect, hasContext := false, false, false
			ast.Inspect(f.Type.Params, func(child ast.Node) bool {
				if _, ok := child.(*ast.ChanType); ok {
					hasChan = true
				}
				return true
			})
			ast.Inspect(f.Body, func(child ast.Node) bool {
				switch x := child.(type) {
				case *ast.ChanType:
					hasChan = true
				case *ast.SelectStmt:
					hasSelect = true
				case *ast.SelectorExpr:
					if x.Sel.Name == "Done" {
						hasContext = true
					}
				}
				return true
			})
			if hasChan && !hasSelect && !hasContext {
				pass.Reportf(f.Pos(), "channel pipeline should provide cancellation handling")
			}
			return true
		})
	}
	return nil
}
