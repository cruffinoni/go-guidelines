package sample

import "fmt"

const badFormat = "failed: %v"
const goodFormat = "failed: %w"

func a(err error) error {
	if err != nil {
		panic(err) // want "wrapped error"
	}
	return nil
}

func b(err error) error {
	return fmt.Errorf("failed: %v", err) // want "with %w"
}

func b2(parseErr error) error {
	return fmt.Errorf("failed: %v", parseErr) // want "with %w"
}

func c(err error) error {
	return fmt.Errorf("failed: %w", err)
}

func d(otherErr error) {
	panic(otherErr) // want "wrapped error"
}

func e(err error) error {
	return fmt.Errorf("progress 100%%w: %v", err) // want "with %w"
}

func f(err error) error {
	return fmt.Errorf("indexed: %[1]w", err)
}

func g(err error) error {
	return fmt.Errorf(badFormat, err) // want "with %w"
}

func h(err error) error {
	return fmt.Errorf(goodFormat, err)
}
