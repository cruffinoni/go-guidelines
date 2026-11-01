package sample

import (
	"errors"
	"fmt"
)

func a() error {
	return errors.New("Bad thing.") // want "capitalized"
}

func b() error {
	return errors.New("bad thing.") // want "punctuation"
}

func c(err error) error {
	return fmt.Errorf("Bad: %w", err) // want "capitalized"
}

func d(err error) error {
	return fmt.Errorf("bad: %w", err)
}

func e() error {
	return errors.New("Bad thing") // want "capitalized"
}

func f() error {
	return errors.New("bad thing")
}

func g() error {
	return fmt.Errorf("%s", "bad")
}
