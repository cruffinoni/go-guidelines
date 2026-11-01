// Package lint defines the contract shared by the analysis engine, the rule
// catalog, and rule implementations.
package lint

import (
	"cmp"
	"fmt"
	"go/token"
)

// Severity ranks how strongly a finding should affect a scan result.
// The zero value is invalid so that rules must declare a severity explicitly.
type Severity uint8

const (
	SeverityInfo Severity = iota + 1
	SeverityWarning
	SeverityError
)

// String returns the lower-case severity name used in reports.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return fmt.Sprintf("Severity(%d)", uint8(s))
	}
}

// MarshalText encodes s as its report name and rejects undeclared severities.
func (s Severity) MarshalText() ([]byte, error) {
	if s < SeverityInfo || s > SeverityError {
		return nil, fmt.Errorf("invalid severity %d", uint8(s))
	}
	return []byte(s.String()), nil
}

// Meta is the single in-code source of truth describing a rule.
type Meta struct {
	// ID is the stable key users enable, disable, and configure, e.g. "GBP001".
	ID string
	// Name is a short kebab-case label, e.g. "no-dot-imports".
	Name string
	// Summary is a one-line description shown in rule listings.
	Summary string
	// Description is the rationale and guidance rendered in Markdown rule docs.
	Description string
	// Severity is attached to every finding the rule reports.
	Severity Severity
	// NeedsTypes requests type information; the engine type-checks packages
	// only when at least one active rule sets it.
	NeedsTypes bool
}

// Rule detects one guideline violation. Implementations must not retain the
// Pass or mutate anything reachable from it other than through Report.
type Rule interface {
	Meta() Meta
	// Check analyzes one package and reports diagnostics through pass.Report.
	// A non-nil error signals an operational failure, not a violation.
	Check(pass *Pass) error
}

// Diagnostic is a violation reported by a rule at a source range.
type Diagnostic struct {
	Pos     token.Pos
	End     token.Pos // optional
	Message string
}

// Finding is a diagnostic resolved to a file position and attributed to a rule.
type Finding struct {
	RuleID   string   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
}

// NewFinding resolves d against fset and attributes it to the rule described by m.
func NewFinding(m Meta, fset *token.FileSet, d Diagnostic) Finding {
	position := fset.Position(d.Pos)
	return Finding{
		RuleID:   m.ID,
		Severity: m.Severity,
		Message:  d.Message,
		File:     position.Filename,
		Line:     position.Line,
		Column:   position.Column,
	}
}

// CompareFindings orders findings by file, line, column, rule ID, and message.
// It is the single ordering that makes concurrent analysis output deterministic.
func CompareFindings(a, b Finding) int {
	return cmp.Or(
		cmp.Compare(a.File, b.File),
		cmp.Compare(a.Line, b.Line),
		cmp.Compare(a.Column, b.Column),
		cmp.Compare(a.RuleID, b.RuleID),
		cmp.Compare(a.Message, b.Message),
	)
}
