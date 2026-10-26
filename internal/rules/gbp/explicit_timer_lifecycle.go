package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ExplicitTimerLifecycle flags timers and tickers whose lifecycle is
// unmanaged.
type ExplicitTimerLifecycle struct{}

// Meta returns the GBP013 identity, severity, and documentation.
func (ExplicitTimerLifecycle) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP013",
		Name:    "explicit-timer-lifecycle",
		Summary: "Flags timers and tickers whose lifecycle is unmanaged.",
		Description: "Reports `time.AfterFunc` calls, and reports `time.NewTicker`/`time.NewTimer` " +
			"results assigned to a variable with no visible `.Stop()` call in the same function. A " +
			"ticker or timer left running leaks its internal goroutine; manage its cancellation " +
			"explicitly and call `Stop`.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports unmanaged timers and tickers in pass.
func (ExplicitTimerLifecycle) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok && callName(c.Fun) == "time.AfterFunc" {
				pass.Reportf(c.Pos(), "time.AfterFunc obscures timer lifecycle; manage timer cancellation explicitly")
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok || f.Body == nil {
				return true
			}
			ast.Inspect(f.Body, func(child ast.Node) bool {
				assign, ok := child.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, rhs := range assign.Rhs {
					call, ok := rhs.(*ast.CallExpr)
					if !ok || (callName(call.Fun) != "time.NewTicker" && callName(call.Fun) != "time.NewTimer") || i >= len(assign.Lhs) {
						continue
					}
					ident, ok := assign.Lhs[i].(*ast.Ident)
					if !ok {
						continue
					}
					stopped := false
					ast.Inspect(f.Body, func(candidate ast.Node) bool {
						c, ok := candidate.(*ast.CallExpr)
						if !ok {
							return true
						}
						selector, ok := c.Fun.(*ast.SelectorExpr)
						if !ok || selector.Sel.Name != "Stop" {
							return true
						}
						if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == ident.Name {
							stopped = true
						}
						return true
					})
					if !stopped {
						pass.Reportf(ident.Pos(), "ticker or timer is created without a visible Stop call")
					}
				}
				return true
			})
			return true
		})
	}
	return nil
}
