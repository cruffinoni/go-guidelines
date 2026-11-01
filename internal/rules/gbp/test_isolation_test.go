package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestTestIsolation(t *testing.T) {
	linttest.Run(t, TestIsolation{}, "testdata/test_isolation")
}
