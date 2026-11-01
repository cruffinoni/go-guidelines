package harness

// Claude returns the harness that keeps a managed block in CLAUDE.md.
func Claude() Harness {
	return markdownBlock{name: "claude", file: "CLAUDE.md"}
}
