package sample

import "time"

var m = map[string]int{"a": 1}

var list = []int{1, 2, 3}

func TestA(t *T) {
	time.Sleep(time.Second) // want "timing-dependent"
}

func TestB(t *T) {
	<-time.After(time.Second) // want "timing-dependent"
}

func TestC(t *T) {
	<-time.Tick(time.Second) // want "timing-dependent"
}

func TestD(t *T) {
	_ = time.NewTimer(time.Second) // want "timing-dependent"
}

func TestE(t *T) {
	_ = time.NewTicker(time.Second) // want "timing-dependent"
}

func TestF(t *T) {
	time.AfterFunc(time.Second, func() {}) // want "timing-dependent"
}

func TestMapIteration(t *T) {
	for k := range m {
		_ = k
	}
}

func TestSliceIteration(t *T) {
	for _, v := range list {
		_ = v
	}
}
