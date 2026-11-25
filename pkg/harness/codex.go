package harness

// newCodex returns the OpenAI Codex CLI harness. Its user-level file is
// AGENTS.md in $CODEX_HOME (default ~/.codex), and its project-level file is
// ./AGENTS.md.
func newCodex() Harness {
	return instructionFile{
		name:      "codex",
		userBase:  inEnvOrHome("CODEX_HOME", ".codex"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
