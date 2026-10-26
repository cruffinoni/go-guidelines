package sample

import "testing"

func TestSomething(t *testing.T) {}

func Helper(t *testing.T) {} // want "exported function or method is missing a documentation comment"
