package gcm

import (
	"go/ast"
	"iter"
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// exportedFuncs iterates the exported function declarations in file, skipping
// TestXxx functions when file is a test file. Rules filter further by
// receiver, name prefix, or documentation as needed.
func exportedFuncs(pass *lint.Pass, file *ast.File) iter.Seq[*ast.FuncDecl] {
	isTest := pass.IsTestFile(file)
	return func(yield func(*ast.FuncDecl) bool) {
		for _, decl := range file.Decls {
			f, ok := decl.(*ast.FuncDecl)
			if !ok || !ast.IsExported(f.Name.Name) {
				continue
			}
			if isTest && strings.HasPrefix(f.Name.Name, "Test") {
				continue
			}
			if !yield(f) {
				return
			}
		}
	}
}

// hasExampleOutput reports whether example's body contains an Output or
// Unordered output comment, as required for go test to run it.
func hasExampleOutput(file *ast.File, example *ast.FuncDecl) bool {
	if example.Body == nil {
		return false
	}
	for _, comments := range file.Comments {
		if comments.Pos() <= example.Body.Lbrace || comments.End() >= example.Body.Rbrace {
			continue
		}
		text := comments.Text()
		if strings.Contains(text, "Output:") || strings.Contains(text, "Unordered output:") {
			return true
		}
	}
	return false
}

// tautologicalDoc reports whether doc reduces to "<name> returns <words of
// name>", or for a GetX name to "<name> returns <x>", ignoring case and a
// trailing period.
func tautologicalDoc(name, doc string) bool {
	text := strings.ToLower(strings.TrimSpace(doc))
	text = strings.TrimSuffix(text, ".")
	lowerName := strings.ToLower(name)
	words := func(value string) string {
		var result strings.Builder
		for i, r := range value {
			if i > 0 && r >= 'A' && r <= 'Z' {
				result.WriteByte(' ')
			}
			result.WriteString(strings.ToLower(string(r)))
		}
		return result.String()
	}
	nameWords := words(name)
	if text == lowerName+" returns "+nameWords || text == lowerName+" returns the "+nameWords {
		return true
	}
	if strings.HasPrefix(name, "Get") {
		target := words(name[3:])
		return text == lowerName+" returns "+target || text == lowerName+" returns the "+target
	}
	return false
}
