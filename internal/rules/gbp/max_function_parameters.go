package gbp

import (
	"go/ast"
	"go/token"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// MaxFunctionParameters flags functions with too many parameters.
type MaxFunctionParameters struct{}

// maxFunctionParameters is the largest accepted parameter count.
const maxFunctionParameters = 5

// Meta returns the GBP023 identity, severity, and documentation.
func (MaxFunctionParameters) Meta() lint.Meta {
	return lint.Meta{
		ID:      "GBP023",
		Name:    "max-function-parameters",
		Summary: "Flags functions with more than five parameters.",
		Description: "Reports function declarations, function literals, and interface methods with more " +
			"than five parameters. Long parameter lists are hard to read and easy to misorder; group " +
			"related values into a struct or use functional options for optional settings. Receivers are " +
			"not counted.",
		Severity: lint.SeverityWarning,
	}
}

// parameterCount returns the number of parameters in list, counting each
// grouped name and each unnamed parameter separately.
func parameterCount(list *ast.FieldList) int {
	if list == nil {
		return 0
	}
	count := 0
	for _, field := range list.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

// reportTooManyParameters reports a parameter list that exceeds the limit.
func reportTooManyParameters(pass *lint.Pass, params *ast.FieldList, pos token.Pos) {
	if count := parameterCount(params); count > maxFunctionParameters {
		pass.Reportf(pos, "function has %d parameters; keep functions to %d or fewer", count, maxFunctionParameters)
	}
}

// Check reports functions with more than maxFunctionParameters parameters.
func (MaxFunctionParameters) Check(pass *lint.Pass) error {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				reportTooManyParameters(pass, x.Type.Params, x.Pos())
			case *ast.FuncLit:
				reportTooManyParameters(pass, x.Type.Params, x.Pos())
			case *ast.InterfaceType:
				for _, method := range x.Methods.List {
					if ft, ok := method.Type.(*ast.FuncType); ok {
						reportTooManyParameters(pass, ft.Params, method.Pos())
					}
				}
			}
			return true
		})
	}
	return nil
}
