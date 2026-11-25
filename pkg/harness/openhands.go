package harness

// newOpenHands returns the OpenHands harness. Its user-level file is
// ~/.openhands/microagents/gg-lint.md, and its project-level file is
// ./.openhands/microagents/repo.md.
func newOpenHands() Harness {
	return instructionFile{
		name:      "openhands",
		userBase:  inHome(".openhands"),
		userFile:  "microagents/gg-lint.md",
		localFile: ".openhands/microagents/repo.md",
	}
}
