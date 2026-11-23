package harness

// newDevin returns the Devin harness. Its project-level file is ./AGENTS.md;
// it has no user-level file.
func newDevin() Harness {
	return instructionFile{
		name:      "devin",
		localFile: "AGENTS.md",
	}
}
