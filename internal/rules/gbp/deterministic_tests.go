package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// DeterministicTests flags timing-dependent test code.
type DeterministicTests struct{}

// timingFuncs are the time-package calls that make tests timing-dependent.
var timingFuncs = []string{"Sleep", "After", "Tick", "NewTimer", "NewTicker", "AfterFunc"}

// Meta returns the GBP015 identity, severity, and documentation.
func (DeterministicTests) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP015",
		Name:    "deterministic-tests",
		Summary: "Flags timing-dependent test code.",
		Description: "In `_test.go` files, reports timing APIs such as `time.Sleep`, `time.After`, " +
			"`time.Tick`, `time.NewTimer`, `time.NewTicker`, and `time.AfterFunc`. Timing makes tests " +
			"flaky; use `testing/synctest` for time-based code, or synchronize explicitly on the event " +
			"the test waits for.",
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// Check reports timing-dependent test code in pass.
func (DeterministicTests) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		if !pass.IsTestFile(file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, name := range timingFuncs {
				if stdlibFunc(pass, call, "time", name) {
					pass.Reportf(call.Pos(), "time.%s makes tests timing-dependent; use testing/synctest or synchronize explicitly", name)
					break
				}
			}
			return true
		})
	}
	return nil
}
