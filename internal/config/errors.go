package config

import (
	"errors"
	"fmt"
)

// Sentinel errors identify configuration failures.
var (
	ErrMultipleDocuments = errors.New("multiple YAML documents are not supported")
	ErrDecode            = errors.New("invalid configuration")
)

// FileError attributes a configuration failure to one file.
type FileError struct {
	Path string
	Err  error
}

// Error returns a string representation of the configuration error.
func (e *FileError) Error() string { return fmt.Sprintf("config %s: %v", e.Path, e.Err) }

// Unwrap returns the underlying configuration error.
func (e *FileError) Unwrap() error { return e.Err }

// DecodeError marks a YAML decoding failure while preserving the underlying
// parser error and its diagnostic text.
type DecodeError struct {
	Err error
}

// Error returns a string representation of the YAML decode error.
func (e *DecodeError) Error() string { return e.Err.Error() }

// Unwrap returns the YAML parser error.
func (e *DecodeError) Unwrap() error { return e.Err }

// Is reports whether target is ErrDecode or wraps the underlying parser error.
func (e *DecodeError) Is(target error) bool {
	return target == ErrDecode || errors.Is(e.Err, target)
}
