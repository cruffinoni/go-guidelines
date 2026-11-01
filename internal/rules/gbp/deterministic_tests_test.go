package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestDeterministicTests(t *testing.T) {
	linttest.Run(t, DeterministicTests{}, "testdata/deterministic_tests")
}
