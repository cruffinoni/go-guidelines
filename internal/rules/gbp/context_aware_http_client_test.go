package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestContextAwareHTTPClient(t *testing.T) {
	linttest.Run(t, ContextAwareHTTPClient{}, "testdata/context_aware_http_client")
}
