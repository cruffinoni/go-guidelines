package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestNoTautologicalDoc(t *testing.T) {
	linttest.Run(t, gcm.NoTautologicalDoc{}, "testdata/no_tautological_doc")
}
