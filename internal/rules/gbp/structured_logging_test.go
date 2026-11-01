package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestStructuredLogging(t *testing.T) {
	linttest.Run(t, StructuredLogging{}, "testdata/structured_logging")
}
