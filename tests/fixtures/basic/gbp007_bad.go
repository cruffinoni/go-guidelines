package util

type Doer interface {
	M1()
	M2()
	M3()
	M4()
	M5()
	M6()
	M7()
	M8()
	M9()
	M10()
}

type doerImpl struct{}

func NewDoer() Doer {
	return &doerImpl{}
}
