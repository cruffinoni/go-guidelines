package harness

// newGoose returns the goose harness. Its user-level file is .goosehints in
// $XDG_CONFIG_HOME/goose (default ~/.config/goose), and its project-level
// file is ./.goosehints.
func newGoose() Harness {
	return instructionFile{
		name:      "goose",
		userBase:  inXDGConfig("goose"),
		userFile:  ".goosehints",
		localFile: ".goosehints",
	}
}
