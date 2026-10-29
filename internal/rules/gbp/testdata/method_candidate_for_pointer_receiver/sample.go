package sample

type Sample struct{}

type helper struct{}

func UseSample(s *Sample) {} // want "may belong as a method"

func UseHelper(h *helper) {}

func NoParams() {}
