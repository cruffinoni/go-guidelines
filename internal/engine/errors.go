package engine

import (
	"fmt"
)

// PanicError reports a rule panic without stopping other rules.
type PanicError struct {
	RuleID string
	Value  any
	Stack  []byte
}

// Error returns a string representation of the panic.
func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v\n%s", e.Value, e.Stack)
}

// RuleError attributes an operational rule failure to one package.
type RuleError struct {
	RuleID    string
	PackageID string
	Err       error
}

// Error returns a string representation of the rule error.
func (e *RuleError) Error() string {
	return fmt.Sprintf("rule %s on package %s: %v", e.RuleID, e.PackageID, e.Err)
}

// Unwrap returns the underlying rule error.
func (e *RuleError) Unwrap() error { return e.Err }

// InvalidPositionError reports a rule diagnostic with an invalid position.
type InvalidPositionError struct {
	RuleID    string
	PackageID string
}

// Error returns a string representation of the invalid position error.
func (e *InvalidPositionError) Error() string {
	return fmt.Sprintf("rule %s on package %s: diagnostic has invalid position", e.RuleID, e.PackageID)
}
