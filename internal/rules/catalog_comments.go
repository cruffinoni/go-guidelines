package rules

import (
	"go/ast"
	"go/token"
	"strings"
)

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

func analyzeDocumentationRule(id string, file *ast.File, isTest bool, add func(token.Pos, string)) {
	switch id {
	case "GCM001", "GCM004":
		for _, d := range file.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range g.Specs {
				ts, ok := s.(*ast.TypeSpec)
				if !ok || !ast.IsExported(ts.Name.Name) {
					continue
				}
				if id == "GCM004" && !strings.HasSuffix(ts.Name.Name, "Config") {
					continue
				}
				if id == "GCM001" && strings.HasSuffix(ts.Name.Name, "Config") {
					continue
				}
				if ts.Doc == nil && g.Doc == nil {
					add(ts.Pos(), "exported type is missing a documentation comment")
				}
			}
		}
	case "GCM002", "GCM003", "GCM008", "GCM009":
		for _, d := range file.Decls {
			f, ok := d.(*ast.FuncDecl)
			if !ok || !ast.IsExported(f.Name.Name) {
				continue
			}
			if isTest && strings.HasPrefix(f.Name.Name, "Test") {
				continue
			}
			if id == "GCM003" && f.Recv == nil {
				continue
			}
			if id == "GCM002" && f.Recv != nil {
				continue
			}
			if id == "GCM008" && !strings.HasPrefix(f.Name.Name, "Example") {
				continue
			}
			if id == "GCM008" {
				if !hasExampleOutput(file, f) {
					add(f.Pos(), "example function is missing an Output comment")
				}
				continue
			}
			doc := f.Doc
			if doc == nil {
				if id == "GCM002" || id == "GCM003" {
					add(f.Pos(), "exported function or method is missing a documentation comment")
				}
				continue
			}
			if id == "GCM009" && tautologicalDoc(f.Name.Name, doc.Text()) {
				add(f.Pos(), "documentation comment appears tautological and low-value")
			}
		}
	case "GCM005":
		for _, d := range file.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}
			for _, spec := range g.Specs {
				v, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range v.Names {
					if strings.HasPrefix(name.Name, "Err") && ast.IsExported(name.Name) && v.Doc == nil && g.Doc == nil {
						add(name.Pos(), "exported error variable is missing a documentation comment")
					}
				}
			}
		}
	case "GCM006", "GCM007":
		max := 120
		for _, group := range file.Comments {
			for _, c := range group.List {
				line := strings.TrimSpace(c.Text)
				if id == "GCM006" && (strings.Contains(line, "**") || strings.Contains(line, "](") || strings.HasPrefix(line, "// ##")) {
					add(c.Pos(), "avoid Markdown formatting in Go comments")
				}
				if id == "GCM007" && len(c.Text) > max {
					add(c.Pos(), "comment line exceeds 120 characters")
				}
			}
		}
	}
}
