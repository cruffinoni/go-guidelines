package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestNoNilCheckForRequiredDependency(t *testing.T) {
	linttest.Run(t, NoNilCheckForRequiredDependency{}, "testdata/no_nil_check_for_required_dependency")
}
