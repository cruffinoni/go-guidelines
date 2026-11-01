package fixture_test

import (
	"testing"

	"example.com/fixture"
)

func TestLibExternal(t *testing.T) {
	if fixture.Lib() != 42 {
		t.Fatal("unexpected value")
	}
}
