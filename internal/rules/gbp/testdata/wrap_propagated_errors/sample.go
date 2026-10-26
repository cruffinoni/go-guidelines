package sample

import "fmt"

func a(err error) error {
	if err != nil {
		panic(err) // want "wrapped error"
	}
	return nil
}

func b(err error) error {
	return fmt.Errorf("failed: %v", err) // want "with %w"
}

func c(err error) error {
	return fmt.Errorf("failed: %w", err)
}

func d(otherErr error) {
	panic(otherErr)
}
