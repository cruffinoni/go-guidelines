package sample

var _ = []int{} // want "empty slice literal"
var _ = []int{1, 2}
var _ = [3]int{}
