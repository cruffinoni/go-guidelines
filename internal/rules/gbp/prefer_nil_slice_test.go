package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestPreferNilSlice(t *testing.T) {
	linttest.Run(t, PreferNilSlice{}, "testdata/prefer_nil_slice")
}
