package util

func BuildOrder() {
	resolve()
}

func helper() {}

func RecA() {
	RecB()
}

func resolve() {}

func RecB() {
	RecA()
}
