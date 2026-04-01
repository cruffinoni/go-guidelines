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

func (d *doerImpl) M1()  {}
func (d *doerImpl) M2()  {}
func (d *doerImpl) M3()  {}
func (d *doerImpl) M4()  {}
func (d *doerImpl) M5()  {}
func (d *doerImpl) M6()  {}
func (d *doerImpl) M7()  {}
func (d *doerImpl) M8()  {}
func (d *doerImpl) M9()  {}
func (d *doerImpl) M10() {}

func NewDoer() Doer {
	return &doerImpl{}
}
