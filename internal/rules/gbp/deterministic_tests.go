package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// DeterministicTests flags timing- and map-order-dependent test code.
type DeterministicTests struct{}

// Meta returns the GBP015 identity, severity, and documentation.
func (DeterministicTests) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP015",
		Name:    "deterministic-tests",
		Summary: "Flags timing- and map-order-dependent test code.",
		Description: "In `_test.go` files, reports `time.Sleep` calls and reports `range` loops over a " +
			"variable known to hold a map literal. Sleeping makes a test's timing flaky, and ranging over " +
			"a map iterates in randomized order; synchronize on a deadline and sort map-derived values " +
			"before asserting on them.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports timing- and map-order-dependent test code in pass.
func (DeterministicTests) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		if !pass.IsTestFile(file) {
			continue
		}
		mapNames := make(map[string]bool)
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				if _, ok := x.Type.(*ast.MapType); ok {
					for _, name := range x.Names {
						mapNames[name.Name] = true
					}
				}
				for i, value := range x.Values {
					if lit, ok := value.(*ast.CompositeLit); ok {
						if _, ok := lit.Type.(*ast.MapType); ok && i < len(x.Names) {
							mapNames[x.Names[i].Name] = true
						}
					}
				}
			case *ast.AssignStmt:
				for i, value := range x.Rhs {
					if lit, ok := value.(*ast.CompositeLit); ok {
						if _, ok := lit.Type.(*ast.MapType); ok && i < len(x.Lhs) {
							if name, ok := x.Lhs[i].(*ast.Ident); ok {
								mapNames[name.Name] = true
							}
						}
					}
				}
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				if callName(x.Fun) == "time.Sleep" {
					pass.Reportf(x.Pos(), "time.Sleep makes tests timing-dependent; synchronize with a deadline")
				}
			case *ast.RangeStmt:
				if name, ok := x.X.(*ast.Ident); ok && mapNames[name.Name] {
					pass.Reportf(x.For, "test iterates over a map with nondeterministic iteration order")
				}
			}
			return true
		})
	}
	return nil
}
