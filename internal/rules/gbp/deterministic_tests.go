package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// DeterministicTests flags timing-dependent test code.
type DeterministicTests struct{}

// Meta returns the GBP015 identity, severity, and documentation.
func (DeterministicTests) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP015",
		Name:    "deterministic-tests",
		Summary: "Flags timing-dependent test code.",
		Description: "In `_test.go` files, reports `time.Sleep` calls. Sleeping makes a test's timing flaky; " +
			"use `testing/synctest` for time-based code, or synchronize explicitly on the event the test " +
			"waits for.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports timing-dependent test code in pass.
func (DeterministicTests) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		if !pass.IsTestFile(file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok && callName(c.Fun) == "time.Sleep" {
				pass.Reportf(c.Pos(), "time.Sleep makes tests timing-dependent; use testing/synctest or synchronize explicitly")
			}
			return true
		})
	}
	return nil
}
