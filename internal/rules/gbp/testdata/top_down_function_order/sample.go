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

func shadowedByParam(f func()) {
	f()
}

func shadowedByLocal() {
	g := func() {}
	g()
}

func f() {
}

func g() {
}

type T struct{}

func (t T) first() {
	t.second() // want "called before its declaration"
}

func (t T) second() {
}

func (t T) m1() {
	t.m2()
}

func (t T) m2() {
	t.m1()
}

func (t T) callsLater() {
	later() // want "called before its declaration"
}

func later() {
}

func useT(t T) {
	t.third() // want "called before its declaration"
}

func (T) third() {
}
