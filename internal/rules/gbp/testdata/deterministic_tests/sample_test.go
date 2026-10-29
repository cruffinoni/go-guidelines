package sample

import "time"

var m = map[string]int{"a": 1}

var list = []int{1, 2, 3}

func TestA(t *T) {
	time.Sleep(time.Second) // want "timing-dependent"
}

func TestB(t *T) {
	for k := range m {
		_ = k
	}
}

func TestC(t *T) {
	for _, v := range list {
		_ = v
	}
}
