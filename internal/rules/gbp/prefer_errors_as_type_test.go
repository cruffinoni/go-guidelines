package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestPreferErrorsAsType(t *testing.T) {
	linttest.Run(t, PreferErrorsAsType{}, "testdata/prefer_errors_as_type")
}
