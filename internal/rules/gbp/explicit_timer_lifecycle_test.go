package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestExplicitTimerLifecycle(t *testing.T) {
	linttest.Run(t, ExplicitTimerLifecycle{}, "testdata/explicit_timer_lifecycle")
}
