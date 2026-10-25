package rules

import "testing"

func TestBuiltInPortfolioIncludesRetainedRulesAndExcludesRemovedRules(t *testing.T) {
	got := BuiltInRules()
	want := []string{
		"GBP001", "GBP002", "GBP003", "GBP004", "GBP005", "GBP006",
		"GBP008", "GBP009", "GBP010", "GBP011", "GBP013", "GBP014",
		"GBP015", "GBP016", "GBP017", "GBP018", "GBP019", "GBP021",
		"GCM001", "GCM002", "GCM003", "GCM004", "GCM005", "GCM006",
		"GCM007", "GCM008", "GCM009",
	}
	if len(got) != len(want) {
		t.Fatalf("BuiltInRules() returned %d rules, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID() != id {
			t.Errorf("BuiltInRules()[%d].ID() = %q, want %q", i, got[i].ID(), id)
		}
	}
}

func TestGBP016FlagsInitAsDiscouraged(t *testing.T) {
	findings, err := (GBP016{}).Analyze("sample.go", []byte("package sample\nfunc init() {}\n"))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 1 || findings[0].RuleID != "GBP016" {
		t.Fatalf("findings = %#v, want one GBP016 finding", findings)
	}
}

func TestGBP015FindsMapIterationInTestFunctions(t *testing.T) {
	source := []byte("package sample\nfunc TestMap(t *testing.T) { values := map[string]int{\"a\": 1}; for key := range values { _ = key } }\n")
	findings, err := (catalogRule{id: "GBP015", severity: SeverityWarning}).Analyze("sample_test.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %#v, want one warning for nondeterministic map iteration", findings)
	}
}

func TestGBP013FindsTickerWithoutVisibleStop(t *testing.T) {
	source := []byte("package sample\nimport \"time\"\nfunc run() { ticker := time.NewTicker(time.Second); _ = ticker }\n")
	findings, err := (catalogRule{id: "GBP013", severity: SeverityWarning}).Analyze("sample.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %#v, want one warning for an unstopped ticker", findings)
	}
}

func TestGBP021AllowsContiguousMutuallyRecursiveFunctions(t *testing.T) {
	source := []byte("package sample\nfunc first(n int) { if n > 0 { second(n-1) } }\nfunc second(n int) { if n > 0 { first(n-1) } }\n")
	findings, err := (catalogRule{id: "GBP021", severity: SeverityWarning}).Analyze("sample.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want mutually recursive adjacent declarations accepted", findings)
	}
}

func TestGCM009DoesNotFlagMeaningfulDocumentation(t *testing.T) {
	source := []byte("package sample\n// GetName returns the configured account name.\nfunc GetName() string { return \"x\" }\n")
	findings, err := (GCM009{}).Analyze("sample.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want no finding for meaningful documentation", findings)
	}
}

func TestGCM009IsInfoOnly(t *testing.T) {
	findings, err := (GCM009{}).Analyze("sample.go", []byte("package sample\n// GetName returns the name.\nfunc GetName() string { return \"\" }\n"))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %#v, want one finding", findings)
	}
	if findings[0].Severity != SeverityInfo {
		t.Fatalf("severity = %q, want info", findings[0].Severity)
	}
}

func TestGCM008RecognizesOutputInExampleBody(t *testing.T) {
	source := []byte("package sample\n// ExampleThing demonstrates the feature.\nfunc ExampleThing() {\n	println(\"ok\")\n	// Output: ok\n}\n")
	findings, err := (catalogRule{id: "GCM008", severity: SeverityWarning}).Analyze("sample_test.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want no finding for an example with output", findings)
	}
}

func TestGCM008RecognizesUnorderedOutputInExampleBody(t *testing.T) {
	source := []byte("package sample\nfunc ExampleThing() {\n	// Unordered output:\n	// b\n	// a\n}\n")
	findings, err := (catalogRule{id: "GCM008", severity: SeverityWarning}).Analyze("sample_test.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want no finding for an unordered-output example", findings)
	}
}

func TestGCM008FlagsExampleWithoutOutputComment(t *testing.T) {
	source := []byte("package sample\nfunc ExampleThing() { println(\"missing\") }\n")
	findings, err := (catalogRule{id: "GCM008", severity: SeverityWarning}).Analyze("sample_test.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 1 || findings[0].RuleID != "GCM008" {
		t.Fatalf("findings = %#v, want one GCM008 finding for missing output", findings)
	}
}

func TestGCM006AllowsGoDocHeading(t *testing.T) {
	source := []byte("package sample\n// # Usage\nfunc f() {}\n")
	findings, err := (catalogRule{id: "GCM006", severity: SeverityWarning}).Analyze("sample.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want no finding for a supported Go doc heading", findings)
	}
}

func TestGCM006RejectsMultiHashHeading(t *testing.T) {
	source := []byte("package sample\n// ## Usage\nfunc f() {}\n")
	findings, err := (catalogRule{id: "GCM006", severity: SeverityWarning}).Analyze("sample.go", source)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %#v, want one finding for unsupported multi-hash heading", findings)
	}
}
