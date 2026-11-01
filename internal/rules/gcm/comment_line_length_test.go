package gcm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func runSource(t *testing.T, src string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	linttest.Run(t, gcm.CommentLineLength{}, dir)
}

func TestCommentLineLength(t *testing.T) {
	linttest.Run(t, gcm.CommentLineLength{}, "testdata/comment_line_length")
}

func TestCommentLineLengthCountsPhysicalLinesAndRunes(t *testing.T) {
	tests := map[string]string{
		"physical lines": `package sample

/*
short line one
short line two
short line three
short line four
short line five
short line six
short line seven
short line eight
*/
type ShortBlock struct{}
`,
		"runes": "package sample\n\n// " + strings.Repeat("é", 100) + "\ntype ShortUnicode struct{}\n\n/* " +
			strings.Repeat("é", 121) + ` */ // want "comment line exceeds 120 characters"` + "\ntype LongUnicode struct{}\n",
	}

	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			runSource(t, src)
		})
	}
}
