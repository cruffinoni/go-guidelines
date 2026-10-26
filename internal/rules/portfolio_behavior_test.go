package rules

import (
	"strings"
	"testing"
)

func TestRetainedRulesDetectTheirDocumentedPatterns(t *testing.T) {
	cases := []struct{ id, filename, source string }{
		{"GBP002", "sample.go", "package util\n"},
		{"GBP003", "sample.go", "package p\nfunc f(err error) { panic(err) }\n"},
		{"GBP004", "sample.go", "package p\nimport \"context\"\ntype S struct { ctx context.Context }\n"},
		{"GBP005", "sample.go", "package p\nfunc f() { go work() }\n"},
		{"GBP006", "sample.go", "package p\nimport \"net/http\"\nfunc f() { _, _ = http.Get(\"https://example.test\") }\n"},
		{"GBP006", "sample.go", "package p\nimport \"net/http\"\nfunc f() { _, _ = http.NewRequest(\"GET\", \"/\", nil) }\n"},
		{"GBP006", "sample.go", "package p\nimport \"net/http\"\nfunc f() { resp, _ := http.Get(\"/\"); _ = resp }\n"},
		{"GBP008", "sample.go", "package p\nfunc f() { panic(\"bad\") }\n"},
		{"GBP009", "sample.go", "package p\nvar xs = []int{}\n"},
		{"GBP010", "sample.go", "package p\ntype S struct { Items []string `json:\"items\"` }\n"},
		{"GBP011", "sample.go", "package p\nimport \"log\"\nfunc f() { log.Printf(\"x\") }\n"},
		{"GBP013", "sample.go", "package p\nimport \"time\"\nfunc f() { time.AfterFunc(time.Second, func(){}) }\n"},
		{"GBP014", "sample.go", "package p\nfunc f(ch chan int) { ch <- 1 }\n"},
		{"GBP015", "sample_test.go", "package p\nimport \"time\"\nfunc TestF() { time.Sleep(time.Second) }\n"},
		{"GBP017", "sample.go", "package p\nvar a int\nvar b int\nvar c int\n"},
		{"GBP018", "sample.go", "package p\ntype Thing struct{}\nfunc Do(x *Thing) {}\n"},
		{"GBP019", "sample.go", "package p\ntype Thing struct{}\nfunc NewThing(x *Thing) *Thing { if x == nil { return nil }; return x }\n"},
		{"GBP021", "sample.go", "package p\nfunc first() { second() }\nfunc second() {}\n"},
		{"GCM001", "sample.go", "package p\ntype Exported struct{}\n"},
		{"GCM002", "sample.go", "package p\nfunc Exported() {}\n"},
		{"GCM003", "sample.go", "package p\ntype T struct{}\nfunc (T) Exported() {}\n"},
		{"GCM004", "sample.go", "package p\ntype AppConfig struct{}\n"},
		{"GCM005", "sample.go", "package p\nvar ErrNotFound = errors.New(\"missing\")\n"},
		{"GCM006", "sample.go", "package p\n// **bold** comment\nfunc f() {}\n"},
		{"GCM007", "sample.go", "package p\n// " + strings.Repeat("x", 121) + "\nfunc f() {}\n"},
		{"GCM008", "sample_test.go", "package p\n// ExampleThing demonstrates use.\nfunc ExampleThing() {}\n"},
	}
	byID := make(map[string]Rule)
	for _, rule := range BuiltInRules() {
		byID[rule.ID()] = rule
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			findings, err := byID[tc.id].Analyze(tc.filename, []byte(tc.source))
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if len(findings) == 0 {
				t.Fatalf("Analyze() returned no finding for %s", tc.id)
			}
			for _, finding := range findings {
				if finding.RuleID != tc.id {
					t.Errorf("finding RuleID = %q, want %q", finding.RuleID, tc.id)
				}
			}
		})
	}
}
