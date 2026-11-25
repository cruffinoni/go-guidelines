package harness

// newOpenClaw returns the OpenClaw harness. Its user-level file is
// ~/.openclaw/workspace/AGENTS.md; it has no project-level file.
func newOpenClaw() Harness {
	return instructionFile{
		name:     "openclaw",
		userBase: inHome(".openclaw"),
		userFile: "workspace/AGENTS.md",
	}
}
