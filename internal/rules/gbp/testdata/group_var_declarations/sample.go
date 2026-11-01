package sample

var a = 1 // want "can be grouped"
var b = 2
var c = 3
var d = 4

func f() {}

var e = 5
var g = 6

var query = `
var one = 1
var two = 2
var three = 3
`

/*
var four = 4
var five = 5
var six = 6
*/

var (
	groupedOne = 1
	groupedTwo = 2
)

var multiline = []int{
	1,
	2,
}

func localVars() {
	var a = 1 // want "can be grouped"
	var b = 2
	var c = 3
}

func crossA() { var a = 1 }
func crossB() { var b = 2 }
func crossC() { var c = 3 }
