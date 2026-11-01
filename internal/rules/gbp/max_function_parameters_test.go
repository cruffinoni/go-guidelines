package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestMaxFunctionParameters(t *testing.T) {
	linttest.Run(t, MaxFunctionParameters{}, "testdata/max_function_parameters")
}
