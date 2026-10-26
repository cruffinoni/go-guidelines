// Package rules defines and analyzes the retained Go guideline rule portfolio.
package rules

// Severity represents how strongly a finding should affect a scan result.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding is a stable, location-aware rule diagnostic.
type Finding struct {
	RuleID   string   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
}

// Rule analyzes one Go source file and returns diagnostics without modifying it.
type Rule interface {
	ID() string
	Analyze(filename string, source []byte) ([]Finding, error)
}
