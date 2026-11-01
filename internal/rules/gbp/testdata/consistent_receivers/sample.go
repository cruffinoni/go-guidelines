package sample

type Mixed struct{}

func (Mixed) Value() {} // want "both value and pointer receiver"

func (*Mixed) Pointer() {}

type ValueOnly struct{}

func (ValueOnly) A() {}
func (ValueOnly) B() {}

type PointerOnly struct{}

func (*PointerOnly) A() {}
func (*PointerOnly) B() {}

type Generic[T any] struct{}

func (Generic[T]) Value() {} // want "both value and pointer receiver"

func (*Generic[T]) Pointer() {}
