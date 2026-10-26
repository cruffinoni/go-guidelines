package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestDocumentSliceJSONNilVsEmpty(t *testing.T) {
	linttest.Run(t, DocumentSliceJSONNilVsEmpty{}, "testdata/document_slice_json_nil_vs_empty")
}
