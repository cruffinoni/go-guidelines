package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestNoPanicInControlFlow(t *testing.T) {
	linttest.Run(t, NoPanicInControlFlow{}, "testdata/no_panic_in_control_flow")
}
