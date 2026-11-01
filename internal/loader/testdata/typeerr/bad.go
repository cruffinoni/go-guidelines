package typeerr

// Bad has a deliberate type mismatch used to test Load's warning path.
func Bad() int {
	var x string = 5
	return len(x)
}
