package sample

// T is a receiver type.
type T struct{}

// Method explains itself.
func (T) Method() {}

func (T) Undocumented() {} // want "exported function or method is missing a documentation comment"

func (T) unexported() {}

// PlainFunc must not be affected by GCM003.
func PlainFunc() {}

func PlainUndocumented() {}
