package rules

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type catalogRule struct {
	id       string
	severity Severity
}

func (r catalogRule) ID() string { return r.id }
func (r catalogRule) Analyze(filename string, source []byte) ([]Finding, error) {
	return analyzeCatalogRule(r.id, r.severity, filename, source)
}

type GBP016 struct{}

func (GBP016) ID() string { return "GBP016" }
func (GBP016) Analyze(filename string, source []byte) ([]Finding, error) {
	return analyzeCatalogRule("GBP016", SeverityWarning, filename, source)
}

type GCM009 struct{}

func (GCM009) ID() string { return "GCM009" }
func (GCM009) Analyze(filename string, source []byte) ([]Finding, error) {
	return analyzeCatalogRule("GCM009", SeverityInfo, filename, source)
}

// BuiltInRules returns the retained GBP/GCM portfolio in stable display order.
func BuiltInRules() []Rule {
	ids := []string{"GBP001", "GBP002", "GBP003", "GBP004", "GBP005", "GBP006", "GBP008", "GBP009", "GBP010", "GBP011", "GBP013", "GBP014", "GBP015", "GBP016", "GBP017", "GBP018", "GBP019", "GBP021", "GCM001", "GCM002", "GCM003", "GCM004", "GCM005", "GCM006", "GCM007", "GCM008", "GCM009"}
	rules := make([]Rule, 0, len(ids))
	for _, id := range ids {
		switch id {
		case "GBP001":
			rules = append(rules, GBP001{})
		case "GBP016":
			rules = append(rules, GBP016{})
		case "GCM009":
			rules = append(rules, GCM009{})
		default:
			rules = append(rules, catalogRule{id: id, severity: ruleSeverity(id)})
		}
	}
	return rules
}

func ruleSeverity(id string) Severity {
	switch id {
	case "GBP001", "GBP003":
		return SeverityError
	case "GBP018", "GCM009":
		return SeverityInfo
	default:
		return SeverityWarning
	}
}

func analyzeCatalogRule(id string, severity Severity, filename string, source []byte) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse Go source %q: %w", filename, err)
	}
	findings := make([]Finding, 0)
	add := func(pos token.Pos, message string) {
		p := fset.Position(pos)
		findings = append(findings, Finding{RuleID: id, Severity: severity, Message: message, File: filename, Line: p.Line, Column: p.Column})
	}
	callName := func(expr ast.Expr) string {
		switch x := expr.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			if pkg, ok := x.X.(*ast.Ident); ok {
				return pkg.Name + "." + x.Sel.Name
			}
			return x.Sel.Name
		}
		return ""
	}
	isTest := strings.HasSuffix(filename, "_test.go")
	switch id {
	case "GBP002":
		if file.Name.Name == "util" || file.Name.Name == "utils" || file.Name.Name == "common" || file.Name.Name == "helpers" {
			add(file.Name.Pos(), "avoid vague package names; use a domain-specific name")
		}
		if file.Doc == nil {
			add(file.Name.Pos(), "package is missing package-level documentation")
		}
	case "GBP003", "GBP008":
		ast.Inspect(file, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := callName(c.Fun)
			if id == "GBP003" && name == "panic" && len(c.Args) > 0 {
				if ident, ok := c.Args[0].(*ast.Ident); ok && ident.Name == "err" {
					add(c.Pos(), "avoid panic(err); return a wrapped error")
				}
			}
			if id == "GBP008" && name == "panic" {
				add(c.Pos(), "avoid panic in normal control flow")
			}
			if id == "GBP003" && name == "fmt.Errorf" && len(c.Args) > 1 {
				if lit, ok := c.Args[0].(*ast.BasicLit); ok && !strings.Contains(lit.Value, "%w") {
					for _, arg := range c.Args[1:] {
						if ident, ok := arg.(*ast.Ident); ok && ident.Name == "err" {
							add(c.Pos(), "wrap propagated errors with %w")
						}
					}
				}
			}
			return true
		})
	case "GBP004":
		ast.Inspect(file, func(n ast.Node) bool {
			if st, ok := n.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					if se, ok := field.Type.(*ast.SelectorExpr); ok {
						if pkg, ok := se.X.(*ast.Ident); ok && pkg.Name == "context" && se.Sel.Name == "Context" {
							add(field.Pos(), "do not store context.Context in a struct")
						}
					}
				}
			}
			if ft, ok := n.(*ast.FuncType); ok {
				for i, p := range ft.Params.List {
					if se, ok := p.Type.(*ast.SelectorExpr); ok {
						if pkg, ok := se.X.(*ast.Ident); ok && pkg.Name == "context" && se.Sel.Name == "Context" && i > 0 {
							add(p.Pos(), "context.Context should be the first parameter")
						}
					}
				}
			}
			return true
		})
	case "GBP005":
		ast.Inspect(file, func(n ast.Node) bool {
			if s, ok := n.(*ast.GoStmt); ok {
				add(s.Go, "goroutine lifecycle and error handling must be explicit")
			}
			return true
		})
	case "GBP006":
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				switch callName(c.Fun) {
				case "http.Get":
					add(c.Pos(), "avoid http.Get; use a context-aware request and reusable client")
				case "http.NewRequest":
					add(c.Pos(), "use http.NewRequestWithContext when request cancellation is available")
				}
			}
			return true
		})
	case "GBP009":
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CompositeLit); ok && len(c.Elts) == 0 {
				if a, ok := c.Type.(*ast.ArrayType); ok && a.Len == nil {
					add(c.Pos(), "empty slice literal; prefer nil slice unless non-nil JSON semantics are required")
				}
			}
			return true
		})
	case "GBP010":
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.Field)
			if !ok || f.Tag == nil {
				return true
			}
			tag := strings.Trim(f.Tag.Value, "`")
			if !strings.Contains(tag, "json:") {
				return true
			}
			if _, ok := f.Type.(*ast.ArrayType); !ok {
				return true
			}
			if strings.Contains(tag, "omitempty") || strings.Contains(tag, "json:\"-\"") {
				return true
			}
			add(f.Pos(), "slice JSON field should document nil-versus-empty encoding (consider omitempty)")
			return true
		})
	case "GBP011":
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				name := callName(c.Fun)
				if strings.HasPrefix(name, "log.Print") || strings.HasPrefix(name, "log.Fatal") || strings.HasPrefix(name, "log.Panic") {
					add(c.Pos(), "prefer structured logging over log package print calls")
				}
			}
			return true
		})
	case "GBP013":
		ast.Inspect(file, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok && callName(c.Fun) == "time.AfterFunc" {
				add(c.Pos(), "time.AfterFunc obscures timer lifecycle; manage timer cancellation explicitly")
			}
			return true
		})
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok || f.Body == nil {
				return true
			}
			ast.Inspect(f.Body, func(child ast.Node) bool {
				assign, ok := child.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, rhs := range assign.Rhs {
					call, ok := rhs.(*ast.CallExpr)
					if !ok || (callName(call.Fun) != "time.NewTicker" && callName(call.Fun) != "time.NewTimer") || i >= len(assign.Lhs) {
						continue
					}
					ident, ok := assign.Lhs[i].(*ast.Ident)
					if !ok {
						continue
					}
					stopped := false
					ast.Inspect(f.Body, func(candidate ast.Node) bool {
						c, ok := candidate.(*ast.CallExpr)
						if !ok {
							return true
						}
						selector, ok := c.Fun.(*ast.SelectorExpr)
						if !ok || selector.Sel.Name != "Stop" {
							return true
						}
						if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == ident.Name {
							stopped = true
						}
						return true
					})
					if !stopped {
						add(ident.Pos(), "ticker or timer is created without a visible Stop call")
					}
				}
				return true
			})
			return true
		})
	case "GBP014":
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			hasChan, hasSelect, hasContext := false, false, false
			ast.Inspect(f.Type.Params, func(child ast.Node) bool {
				if _, ok := child.(*ast.ChanType); ok {
					hasChan = true
				}
				return true
			})
			ast.Inspect(f.Body, func(child ast.Node) bool {
				switch x := child.(type) {
				case *ast.ChanType:
					hasChan = true
				case *ast.SelectStmt:
					hasSelect = true
				case *ast.SelectorExpr:
					if x.Sel.Name == "Done" {
						hasContext = true
					}
				}
				return true
			})
			if hasChan && !hasSelect && !hasContext {
				add(f.Pos(), "channel pipeline should provide cancellation handling")
			}
			return true
		})
	case "GBP015":
		if isTest {
			mapNames := make(map[string]bool)
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ValueSpec:
					if _, ok := x.Type.(*ast.MapType); ok {
						for _, name := range x.Names {
							mapNames[name.Name] = true
						}
					}
					for i, value := range x.Values {
						if lit, ok := value.(*ast.CompositeLit); ok {
							if _, ok := lit.Type.(*ast.MapType); ok && i < len(x.Names) {
								mapNames[x.Names[i].Name] = true
							}
						}
					}
				case *ast.AssignStmt:
					for i, value := range x.Rhs {
						if lit, ok := value.(*ast.CompositeLit); ok {
							if _, ok := lit.Type.(*ast.MapType); ok && i < len(x.Lhs) {
								if name, ok := x.Lhs[i].(*ast.Ident); ok {
									mapNames[name.Name] = true
								}
							}
						}
					}
				}
				return true
			})
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					if callName(x.Fun) == "time.Sleep" {
						add(x.Pos(), "time.Sleep makes tests timing-dependent; synchronize with a deadline")
					}
				case *ast.RangeStmt:
					if name, ok := x.X.(*ast.Ident); ok && mapNames[name.Name] {
						add(x.For, "test iterates over a map with nondeterministic iteration order")
					}
				}
				return true
			})
		}
	case "GBP016":
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil && f.Name.Name == "init" {
				add(f.Pos(), "avoid init; prefer explicit setup")
			}
		}
	case "GBP017":
		lines := strings.Split(string(source), "\n")
		run := 0
		start := 0
		tf := fset.File(file.Pos())
		for i, l := range lines {
			trim := strings.TrimSpace(l)
			if strings.HasPrefix(trim, "var ") && !strings.HasPrefix(trim, "var (") {
				if run == 0 {
					start = i + 1
				}
				run++
			} else {
				if run >= 3 && tf != nil {
					add(token.Pos(tf.LineStart(start)), fmt.Sprintf("%d consecutive var declarations can be grouped", run))
				}
				run = 0
			}
		}
		if run >= 3 && tf != nil {
			add(token.Pos(tf.LineStart(start)), fmt.Sprintf("%d consecutive var declarations can be grouped", run))
		}
	case "GBP018":
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok || f.Recv != nil || len(f.Type.Params.List) == 0 {
				return true
			}
			star, ok := f.Type.Params.List[0].Type.(*ast.StarExpr)
			if !ok {
				return true
			}
			if ident, ok := star.X.(*ast.Ident); ok && ast.IsExported(ident.Name) {
				add(f.Pos(), "function taking a struct pointer first may belong as a method")
			}
			return true
		})
	case "GBP019":
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(f.Name.Name, "New") || f.Body == nil {
				return true
			}
			ast.Inspect(f.Body, func(child ast.Node) bool {
				if s, ok := child.(*ast.IfStmt); ok {
					if b, ok := s.Cond.(*ast.BinaryExpr); ok && b.Op == token.EQL {
						if id, ok := b.Y.(*ast.Ident); ok && id.Name == "nil" {
							if x, ok := b.X.(*ast.Ident); ok && x.Name != "err" {
								add(s.Pos(), "constructor nil-checks a required dependency instead of enforcing its contract")
							}
						}
					}
				}
				return true
			})
			return true
		})
	case "GBP021":
		funcs := []*ast.FuncDecl{}
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil {
				funcs = append(funcs, f)
			}
		}
		index := map[string]int{}
		for i, f := range funcs {
			if _, exists := index[f.Name.Name]; !exists {
				index[f.Name.Name] = i
			}
		}
		calls := make([]map[string]bool, len(funcs))
		for i, f := range funcs {
			calls[i] = make(map[string]bool)
			if f.Body != nil {
				ast.Inspect(f.Body, func(n ast.Node) bool {
					if c, ok := n.(*ast.CallExpr); ok {
						if called, ok := c.Fun.(*ast.Ident); ok {
							calls[i][called.Name] = true
						}
					}
					return true
				})
			}
		}
		for i, f := range funcs {
			if f.Body == nil {
				continue
			}
			ast.Inspect(f.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				id, ok := c.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				if j, exists := index[id.Name]; exists && j > i && j != i {
					if j == i+1 && calls[j][f.Name.Name] {
						return true
					}
					add(c.Pos(), "local function is called before its declaration; consider top-down declaration order")
				}
				return true
			})
		}
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
	return findings, nil
}

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
