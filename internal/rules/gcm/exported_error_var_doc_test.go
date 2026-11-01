package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestExportedErrorVarDoc(t *testing.T) {
	linttest.Run(t, gcm.ExportedErrorVarDoc{}, "testdata/exported_error_var_doc")
}
