package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestGroupVarDeclarations(t *testing.T) {
	linttest.Run(t, GroupVarDeclarations{}, "testdata/group_var_declarations")
}
