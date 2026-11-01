package sample

import (
	"fmt"
	"testing"
)

func ExampleGood() {
	fmt.Println("hi")
	// Output: hi
}

func ExampleBad() { // want "example function is missing an Output comment"
	fmt.Println("hi")
}

func ExampleUnordered() {
	fmt.Println("a")
	fmt.Println("b")
	// Unordered output: a
	// b
}

func TestHelper(t *testing.T) {}

func unexportedExample() {}
