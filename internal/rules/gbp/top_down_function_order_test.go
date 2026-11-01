package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestTopDownFunctionOrder(t *testing.T) {
	linttest.Run(t, TopDownFunctionOrder{}, "testdata/top_down_function_order")
}
