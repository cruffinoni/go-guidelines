package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestExplicitGoroutineLifecycle(t *testing.T) {
	linttest.Run(t, ExplicitGoroutineLifecycle{}, "testdata/explicit_goroutine_lifecycle")
}
