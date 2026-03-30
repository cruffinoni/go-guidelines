package order

func Build() {
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
