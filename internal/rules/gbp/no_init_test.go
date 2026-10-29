package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestNoInit(t *testing.T) {
	linttest.Run(t, NoInit{}, "testdata/no_init")
}
