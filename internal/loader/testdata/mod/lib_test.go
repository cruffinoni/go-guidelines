package fixture

import "testing"

func TestLib(t *testing.T) {
	if Lib() != 42 {
		t.Fatal("unexpected value")
	}
}
