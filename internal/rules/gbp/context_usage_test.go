package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestContextUsage(t *testing.T) {
	linttest.Run(t, ContextUsage{}, "testdata/context_usage")
}
