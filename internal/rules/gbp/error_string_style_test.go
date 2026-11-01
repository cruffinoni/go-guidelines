package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestErrorStringStyle(t *testing.T) {
	linttest.Run(t, ErrorStringStyle{}, "testdata/error_string_style")
}
