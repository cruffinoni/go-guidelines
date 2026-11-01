package sample

func f() {
	panic("boom") // want "normal control flow"
}

func g() {
	_ = 1
}
