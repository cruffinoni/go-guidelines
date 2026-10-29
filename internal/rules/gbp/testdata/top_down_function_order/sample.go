package sample

func a() {
	b() // want "called before its declaration"
}

func b() {
}

func c() {
	d()
}

func d() {
	c()
}
