package harness

import (
	"fmt"
	"os"
	"path/filepath"
)

// homeDir returns ~/rel.
func homeDir(rel string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, filepath.FromSlash(rel)), nil
}

// inHome returns a base resolver for ~/rel.
func inHome(rel string) func() (string, error) {
	return func() (string, error) { return homeDir(rel) }
}

// inEnvOrHome returns a base resolver for the directory named by env when
// it holds an absolute path, and ~/rel otherwise.
func inEnvOrHome(env, rel string) func() (string, error) {
	return func() (string, error) {
		if dir := os.Getenv(env); filepath.IsAbs(dir) {
			return dir, nil
		}
		return homeDir(rel)
	}
}

// inXDGConfig returns a base resolver for $XDG_CONFIG_HOME/rel when
// XDG_CONFIG_HOME holds an absolute path, and ~/.config/rel otherwise, as
// most CLI tools do on every platform.
func inXDGConfig(rel string) func() (string, error) {
	return func() (string, error) {
		if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
			return filepath.Join(dir, filepath.FromSlash(rel)), nil
		}
		return homeDir(".config/" + rel)
	}
}
