package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestExportedFunctionDoc(t *testing.T) {
	linttest.Run(t, gcm.ExportedFunctionDoc{}, "testdata/exported_function_doc")
}
