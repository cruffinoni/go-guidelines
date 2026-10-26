package rules

import "testing"

func TestGBP001FindsDotImportsAtTheirSourcePosition(t *testing.T) {
	tests := []struct {
		name   string
		source string
		line   int
		column int
	}{
		{
			name:   "standalone import",
			source: "package sample\nimport . \"fmt\"\n",
			line:   2,
			column: 8,
		},
		{
			name:   "grouped import",
			source: "package sample\nimport (\n\t\"errors\"\n\t. \"fmt\"\n)\n",
			line:   4,
			column: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := (GBP001{}).Analyze("sample.go", []byte(tt.source))
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if len(findings) != 1 {
				t.Fatalf("Analyze() returned %d findings, want 1", len(findings))
			}
			got := findings[0]
			if got.RuleID != "GBP001" || got.File != "sample.go" || got.Line != tt.line || got.Column != tt.column {
				t.Fatalf("finding = %#v, want GBP001 at sample.go:%d:%d", got, tt.line, tt.column)
			}
			if got.Severity != SeverityError {
				t.Errorf("severity = %q, want %q", got.Severity, SeverityError)
			}
		})
	}
}

func TestGBP001IgnoresOrdinaryImportsAndTextLookalikes(t *testing.T) {
	source := `package sample
// import . "comment"
import "fmt"
var _ = "import ."
`
	findings, err := (GBP001{}).Analyze("clean.go", []byte(source))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("Analyze() returned findings = %#v, want none", findings)
	}
}

func TestGBP001ReturnsParseErrorForInvalidGo(t *testing.T) {
	_, err := (GBP001{}).Analyze("broken.go", []byte("package sample\nimport (\n"))
	if err == nil {
		t.Fatal("Analyze() error = nil, want parse error")
	}
}
