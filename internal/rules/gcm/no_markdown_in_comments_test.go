package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestNoMarkdownInComments(t *testing.T) {
	linttest.Run(t, gcm.NoMarkdownInComments{}, "testdata/no_markdown_in_comments")
}
