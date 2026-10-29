package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestExportedTypeDoc(t *testing.T) {
	linttest.Run(t, gcm.ExportedTypeDoc{}, "testdata/exported_type_doc")
}
