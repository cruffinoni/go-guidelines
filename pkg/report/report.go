// Package report renders lint findings as text or JSON.
package report

import (
	"encoding/json"
	"fmt"
	"io"
)

// Finding is one reported issue, serialized in the JSON schema below.
type Finding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// Report is the result of a scan: the number of files scanned and the
// findings reported against them.
type Report struct {
	ScannedFiles int
	Findings     []Finding
}

// Format selects how a [Report] is rendered.
type Format string

// Text and JSON are the formats a [Format] can hold; ParseFormat accepts no
// others.
const (
	Text Format = "text"
	JSON Format = "json"
)

// schemaVersion identifies the shape of the JSON document written by
// [JSON.Write]. Bump it when the shape changes incompatibly.
const schemaVersion = "1.0"

// document is the JSON shape written for [JSON].
type document struct {
	SchemaVersion string    `json:"schema_version"`
	ScannedFiles  int       `json:"scanned_files"`
	Findings      []Finding `json:"findings"`
}

// ParseFormat parses s as a [Format]. It returns an error naming s if s is
// neither "text" nor "json".
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case Text, JSON:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unknown format %q (want %q or %q)", s, Text, JSON)
	}
}

// Write renders r to w in format f. For [Text], it writes one line per
// finding and, when there are none, "No findings (N Go files scanned).".
// For [JSON], it writes one JSON object terminated by a newline, with
// findings always encoded as an array, never null.
func (f Format) Write(w io.Writer, r Report) error {
	findings := r.Findings
	if findings == nil {
		findings = make([]Finding, 0)
	}
	switch f {
	case JSON:
		doc := document{SchemaVersion: schemaVersion, ScannedFiles: r.ScannedFiles, Findings: findings}
		return json.NewEncoder(w).Encode(doc)
	case Text:
		for _, finding := range findings {
			fmt.Fprintf(w, "%s:%d:%d: %s [%s] %s\n", finding.File, finding.Line, finding.Column, finding.Severity, finding.RuleID, finding.Message)
		}
		if len(findings) == 0 {
			fmt.Fprintf(w, "No findings (%d Go files scanned).\n", r.ScannedFiles)
		}
		return nil
	default:
		return fmt.Errorf("unknown format %q", f)
	}
}
