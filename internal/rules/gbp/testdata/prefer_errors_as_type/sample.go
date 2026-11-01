package sample

import "errors"

type MyError struct{}

func (MyError) Error() string { return "my error" }

func f(err error) {
	var target *MyError
	_ = errors.As(err, &target) // want "errors.AsType"
}

func g(err error) {
	var target *MyError
	if errors.As(err, &target) { // want "errors.AsType"
		_ = target
	}
}

func h(err error) bool {
	return errors.Is(err, MyError{})
}
