package sample

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
)

func TestBad(t *testing.T) {
	os.Setenv("X", "1")            // want "use t.Setenv"
	os.Unsetenv("X")               // want "t.Cleanup"
	os.Chdir(".")                  // want "t.Chdir"
	_, _ = os.MkdirTemp("", "x")   // want "t.TempDir"
	_, _ = ioutil.TempDir("", "x") // want "t.TempDir"
	_ = context.Background()       // want "t.Context"
}

func TestGood(t *testing.T) {
	t.Setenv("X", "1")
	_ = t.Context()
}

func TestParallel(t *testing.T) {
	t.Setenv("X", "1") // want "t.Parallel"
	t.Parallel()
}

func BenchmarkBad(b *testing.B) {
	for i := 0; i < b.N; i++ { // want "b.Loop"
	}
}
