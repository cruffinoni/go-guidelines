package ok

func first() {} // want "func first"

func second() {} // want `func s\w+`

var notAFunc = 1
