package gbp

import (
	"go/ast"
	"go/types"

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
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// isContextDone reports whether sel is a Done call on a context.Context
// value. It uses type information so that unrelated Done methods, such as
// sync.WaitGroup.Done, are not treated as cancellation.
func isContextDone(pass *lint.Pass, sel *ast.SelectorExpr) bool {
	if sel.Sel.Name != "Done" {
		return false
	}
	t := pass.TypesInfo.TypeOf(sel.X)
	if t == nil {
		return false
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
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
					if isContextDone(pass, x) {
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
