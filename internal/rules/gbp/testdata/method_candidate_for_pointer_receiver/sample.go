package sample

import "net/http"

type Sample struct{}

type helper struct{}

type MyInt int

type Box[T any] struct{ _ T }

func UseSample(s *Sample) {} // want "may belong as a method"

func UseHelper(h *helper) {} // want "may belong as a method"

func UseValue(s Sample) {} // want "may belong as a method"

func UseGeneric(b *Box[int]) {} // want "may belong as a method"

func NoParams() {}

func UseTwo(s *Sample, n int) {}

func UseMyInt(v *MyInt) {}

func UseRequest(r *http.Request) {}

func NewSample(s *Sample) {}

func newSample(s *Sample) {}
