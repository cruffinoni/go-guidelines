package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestChannelCancellation(t *testing.T) {
	linttest.Run(t, ChannelCancellation{}, "testdata/channel_cancellation")
}
