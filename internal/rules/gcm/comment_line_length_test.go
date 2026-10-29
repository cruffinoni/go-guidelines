package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestCommentLineLength(t *testing.T) {
	linttest.Run(t, gcm.CommentLineLength{}, "testdata/comment_line_length")
}
