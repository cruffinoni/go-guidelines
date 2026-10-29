package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestWrapPropagatedErrors(t *testing.T) {
	linttest.Run(t, WrapPropagatedErrors{}, "testdata/wrap_propagated_errors")
}
