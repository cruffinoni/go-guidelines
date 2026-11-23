package harness

// Codex returns the harness that keeps a managed block in AGENTS.md. Its
// user-level file lives in $CODEX_HOME, or ~/.codex when unset.
func Codex() Harness {
	return markdownBlock{
		name:    "codex",
		file:    "AGENTS.md",
		userDir: func() (string, error) { return homeSubdir("CODEX_HOME", ".codex") },
	}
}
