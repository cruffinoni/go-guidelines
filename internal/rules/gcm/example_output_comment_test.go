package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestExampleOutputComment(t *testing.T) {
	linttest.Run(t, gcm.ExampleOutputComment{}, "testdata/example_output_comment")
}
