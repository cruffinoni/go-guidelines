package harness

// Claude returns the harness that keeps a managed block in CLAUDE.md. Its
// user-level file lives in $CLAUDE_CONFIG_DIR, or ~/.claude when unset.
func Claude() Harness {
	return markdownBlock{
		name:    "claude",
		file:    "CLAUDE.md",
		userDir: func() (string, error) { return homeSubdir("CLAUDE_CONFIG_DIR", ".claude") },
	}
}
