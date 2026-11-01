package mismatch

func unexpected() {}

func wrongMessage() {} // want "something else"

var expectedButMissing = 1 // want "never reported"
