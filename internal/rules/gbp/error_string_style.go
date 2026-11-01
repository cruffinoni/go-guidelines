package gbp

import (
	"go/ast"
	"go/token"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// ErrorStringStyle flags error messages that are capitalized or end with
// punctuation.
type ErrorStringStyle struct{}

// Meta returns the GBP025 identity, severity, and documentation.
func (ErrorStringStyle) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP025",
		Name:    "error-string-style",
		Summary: "Flags capitalized error messages and trailing punctuation.",
		Description: "Reports literal messages passed to `errors.New` or `fmt.Errorf` that begin with an " +
			"uppercase letter or end with punctuation or a newline. Error strings are typically embedded " +
			"in other messages, so they should read as lowercase fragments without trailing punctuation.",
		Severity:   lint.SeverityWarning,
		NeedsTypes: true,
	}
}

// errorStringProblem returns a short description of the style problem in s,
// or "" when s is acceptable.
func errorStringProblem(s string) string {
	if s == "" {
		return ""
	}
	first, _ := utf8.DecodeRuneInString(s)
	if unicode.IsUpper(first) {
		return "capitalized"
	}
	last, _ := utf8.DecodeLastRuneInString(s)
	switch last {
	case '.', ':', '!', '?', '\n':
		return "punctuation"
	default:
		return ""
	}
}

// errorCreationMessage returns the literal message passed to errors.New or
// fmt.Errorf, if call is such a call with a literal first argument.
func errorCreationMessage(pass *lint.Pass, call *ast.CallExpr) (string, bool) {
	if !stdlibFunc(pass, call, "errors", "New") && !stdlibFunc(pass, call, "fmt", "Errorf") {
		return "", false
	}
	if len(call.Args) == 0 {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return text, true
}

// Check reports error strings with style problems in pass.
func (ErrorStringStyle) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			text, ok := errorCreationMessage(pass, call)
			if !ok {
				return true
			}
			switch problem := errorStringProblem(text); problem {
			case "capitalized":
				pass.Reportf(call.Pos(), "error string should not be capitalized")
			case "punctuation":
				pass.Reportf(call.Pos(), "error string should not end with punctuation")
			}
			return true
		})
	}
	return nil
}
