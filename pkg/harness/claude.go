package harness

// newClaude returns the Claude Code harness. Its user-level file is
// CLAUDE.md in $CLAUDE_CONFIG_DIR (default ~/.claude), and its project-level
// file is ./CLAUDE.md.
func newClaude() Harness {
	return instructionFile{
		name:      "claude",
		userBase:  inEnvOrHome("CLAUDE_CONFIG_DIR", ".claude"),
		userFile:  "CLAUDE.md",
		localFile: "CLAUDE.md",
	}
}
