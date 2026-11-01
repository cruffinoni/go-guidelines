package sample

type T struct{}

type I interface {
	Six(a, b, c, d, e, f int) // want "6 parameters"
}

func Five(a, b, c, d, e int) {}

func Six(a, b, c, d, e, f int) {} // want "6 parameters"

func Grouped(a, b, c, d, e int) {}

func Variadic(a, b, c, d int, rest ...string) {}

func (T) MethodFive(a, b, c, d, e int) {}

func (T) MethodSix(a, b, c, d, e, f int) {} // want "6 parameters"

var _ = func(a, b, c, d, e, f int) {} // want "6 parameters"
