// Package sample exercises dot imports.
package sample

import (
	"fmt"
	. "strings" // want "dot imports"
)

var _ = fmt.Sprintf
var _ = ToUpper

// import . "commented out, must not fire"
var _ = "import . \"not a real import\""
