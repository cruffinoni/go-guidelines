package harness

// newAmp returns the Amp harness. Its user-level file is AGENTS.md in
// $XDG_CONFIG_HOME/amp (default ~/.config/amp), and its project-level file
// is ./AGENTS.md.
func newAmp() Harness {
	return instructionFile{
		name:      "amp",
		userBase:  inXDGConfig("amp"),
		userFile:  "AGENTS.md",
		localFile: "AGENTS.md",
	}
}
