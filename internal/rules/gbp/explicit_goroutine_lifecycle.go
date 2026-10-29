package gbp

import (
	"go/ast"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ExplicitGoroutineLifecycle flags goroutines started without visible
// lifecycle handling.
type ExplicitGoroutineLifecycle struct{}

// Meta returns the GBP005 identity, severity, and documentation.
func (ExplicitGoroutineLifecycle) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP005",
		Name:    "explicit-goroutine-lifecycle",
		Summary: "Flags goroutines started without visible lifecycle handling.",
		Description: "Reports every `go` statement in the file. A goroutine started this way has no " +
			"visible owner for its errors, panics, or shutdown signal. Use a structured concurrency " +
			"helper, such as `errgroup` or `sync.WaitGroup`, that makes completion and error handling " +
			"explicit.",
		Severity: lint.SeverityWarning,
	}
}

// Check reports every go statement in pass.
func (ExplicitGoroutineLifecycle) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if s, ok := n.(*ast.GoStmt); ok {
				pass.Reportf(s.Go, "goroutine lifecycle and error handling must be explicit")
			}
			return true
		})
	}
	return nil
}
