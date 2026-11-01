package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimRecursiveSuffix(t *testing.T) {
	tests := map[string]string{
		"...":       ".",
		"./...":     "./",
		"./pkg/...": "./pkg/",
		`.\...`:     `.\`,
		`.\pkg\...`: `.\pkg\`,
		"pkg":       "pkg",
		"pkg/...":   "pkg/",
	}
	in := make([]string, 0, len(tests))
	for input := range tests {
		in = append(in, input)
	}
	out := trimRecursiveSuffix(in)
	for i, input := range in {
		assert.Equal(t, tests[input], out[i], "input %q", input)
	}
}
