package sample

import "errors"

// ErrDocumented explains itself.
var ErrDocumented = errors.New("documented")

var ErrUndocumented = errors.New("undocumented") // want "exported error variable is missing a documentation comment"

var errUnexported = errors.New("unexported")

var Other = errors.New("not err-prefixed")

// Grouped covers the doc-on-group case.
var (
	ErrGrouped = errors.New("grouped")
)
