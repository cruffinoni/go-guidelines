package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestExportedMethodDoc(t *testing.T) {
	linttest.Run(t, gcm.ExportedMethodDoc{}, "testdata/exported_method_doc")
}
