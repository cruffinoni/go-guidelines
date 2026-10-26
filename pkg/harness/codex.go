package harness

// Codex returns the harness that keeps a managed block in AGENTS.md.
func Codex() Harness {
	return markdownBlock{name: "codex", file: "AGENTS.md"}
}
