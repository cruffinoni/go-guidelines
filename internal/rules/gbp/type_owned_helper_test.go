package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestTypeOwnedHelper(t *testing.T) {
	linttest.Run(t, TypeOwnedHelper{}, "testdata/type_owned_helper")
}
