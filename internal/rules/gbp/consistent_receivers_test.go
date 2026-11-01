package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestConsistentReceivers(t *testing.T) {
	linttest.Run(t, ConsistentReceivers{}, "testdata/consistent_receivers")
}
