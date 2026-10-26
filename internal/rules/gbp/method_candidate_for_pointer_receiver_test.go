package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestMethodCandidateForPointerReceiver(t *testing.T) {
	linttest.Run(t, MethodCandidateForPointerReceiver{}, "testdata/method_candidate_for_pointer_receiver")
}
