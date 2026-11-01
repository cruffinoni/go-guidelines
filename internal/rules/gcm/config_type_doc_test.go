package gcm_test

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
	"github.com/cruffinoni/go-guidelines/internal/rules/gcm"
)

func TestConfigTypeDoc(t *testing.T) {
	linttest.Run(t, gcm.ConfigTypeDoc{}, "testdata/config_type_doc")
}
