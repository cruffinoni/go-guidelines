package sample

// Documented explains itself.
func Documented() {}

func Undocumented() {} // want "exported function or method is missing a documentation comment"

func unexported() {}

// T is a receiver type.
type T struct{}

func (T) UndocumentedMethod() {}
