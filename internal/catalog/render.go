package catalog

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// WriteTable writes every rule as an aligned table with columns ID, NAME,
// SEVERITY, and SUMMARY.
func (c *Catalog) WriteTable(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	ew := &errWriter{w: tw}
	ew.printf("ID\tNAME\tSEVERITY\tSUMMARY\n")
	for _, r := range c.ordered {
		m := r.Meta()
		ew.printf("%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Severity, m.Summary)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	return ew.err
}

// WriteMarkdown writes every rule as a Markdown document: an index table
// of every rule, followed by one section per rule with its severity and
// description.
func (c *Catalog) WriteMarkdown(w io.Writer) error {
	ew := &errWriter{w: w}
	ew.printf("# gg-lint rules\n\n")
	ew.printf("| ID | Name | Severity | Summary |\n")
	ew.printf("| --- | --- | --- | --- |\n")
	for _, r := range c.ordered {
		m := r.Meta()
		ew.printf("| %s | %s | %s | %s |\n",
			escapeMarkdownCell(m.ID), escapeMarkdownCell(m.Name), m.Severity, escapeMarkdownCell(m.Summary))
	}
	ew.printf("\n")
	for _, r := range c.ordered {
		m := r.Meta()
		ew.printf("## %s: %s\n\n", m.ID, m.Name)
		ew.printf("Severity: %s\n\n", m.Severity)
		ew.printf("%s\n\n", m.Description)
	}
	return ew.err
}

// escapeMarkdownCell escapes the pipe characters that would otherwise break
// a Markdown table cell.
func escapeMarkdownCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// errWriter wraps a writer and records its first write error, so a
// sequence of writes can skip error checks after the first failure.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}
