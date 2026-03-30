package util

import (
	"testing"
	"time"
)

func TestWithSleep(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
}

func TestMapIteration(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	for k := range m {
		_ = k
	}
}
