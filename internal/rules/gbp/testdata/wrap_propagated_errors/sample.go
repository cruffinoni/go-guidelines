package sample

import (
	"fmt"
	"strings"
)

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

// i reproduces the issue: the error is converted to a string before fmt.Errorf.
func i(err error) error {
	detail := err.Error()
	return fmt.Errorf("command failed: %s", detail) // want "with %w"
}

// j passes the stringified error inline.
func j(err error) error {
	return fmt.Errorf("command failed: %s", err.Error()) // want "with %w"
}

// k stringifies the error through fmt.Sprint.
func k(err error) error {
	detail := fmt.Sprint(err)
	return fmt.Errorf("command failed: %v", detail) // want "with %w"
}

// l carries the error text through a string helper.
func l(err error) error {
	detail := strings.TrimSpace(err.Error())
	return fmt.Errorf("command failed: %s", detail) // want "with %w"
}

// m is the real-world shape: the detail is assigned on one branch and the error
// text on another, then passed with %s.
func m(err error, stderr *strings.Builder) error {
	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		detail = err.Error()
	}
	return fmt.Errorf("git %s: %s", stderr.String(), detail) // want "with %w"
}

// n wraps one error while dropping a second one.
func n(err, otherErr error) error {
	return fmt.Errorf("%w and %v", err, otherErr) // want "with %w"
}

// o drops the first error while wrapping the second.
func o(dropped, err error) error {
	return fmt.Errorf("%v: %w", dropped, err) // want "with %w"
}

// p keeps the stringified detail but also wraps the original error.
func p(err error) error {
	detail := err.Error()
	return fmt.Errorf("git: %s: %w", detail, err)
}

// q passes a plain string that never came from an error.
func q() error {
	detail := "static detail"
	return fmt.Errorf("command failed: %s", detail)
}

// r returns the error text without constructing a new error.
func r(err error) string {
	return err.Error()
}

// s builds a string from two errors but wraps neither.
func s(first, second error) error {
	return fmt.Errorf("%s/%s", first.Error(), second.Error()) // want "with %w"
}

func t(err, contextErr error) error {
	detail := err.Error()
	return fmt.Errorf("%w: %s", contextErr, detail) // want "with %w"
}

func u(err error) error {
	detail := err.Error()
	detail = "static detail"
	return fmt.Errorf("%s", detail)
}

func discard(string) string { return "redacted" }

func v(err error) error {
	detail := discard(err.Error())
	return fmt.Errorf("%s", detail)
}
