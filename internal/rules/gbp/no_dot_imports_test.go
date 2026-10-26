package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestNoDotImports(t *testing.T) {
	linttest.Run(t, NoDotImports{}, "testdata/no_dot_imports")
}
