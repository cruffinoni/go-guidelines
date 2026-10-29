// Package config loads and discovers the gg-lint YAML configuration.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// FileName is the name of the project config file looked up at the Git
// worktree root.
const FileName = "gg-lint.yaml"

// File is the on-disk shape of a gg-lint config file.
type File struct {
	Format *string         `yaml:"format"`
	FailOn *string         `yaml:"fail_on"`
	Rules  map[string]bool `yaml:"rules"`
}

// Load decodes the YAML file at path into a [File]. Decoding is strict:
// unknown and duplicate keys are rejected. An empty file decodes to the
// zero [File]. Every error names path; decoding errors also carry the line
// of the offending node.
func Load(path string) (File, error) {
	var file File
	data, err := os.Open(path)
	if err != nil {
		return file, fmt.Errorf("config %s: %w", path, err)
	}
	defer data.Close()

	dec := yaml.NewDecoder(data)
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil && !errors.Is(err, io.EOF) {
		return file, fmt.Errorf("config %s: %w", path, err)
	}
	return file, nil
}

// UserPath returns the path of the user-level config file:
// $XDG_CONFIG_HOME/gg-lint/config.yaml when XDG_CONFIG_HOME is set to an
// absolute path, ~/.config/gg-lint/config.yaml otherwise. Unlike
// [os.UserConfigDir], it uses ~/.config on macOS too, as most CLI tools do.
func UserPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(dir) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user config directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gg-lint", "config.yaml"), nil
}

// Discover returns the first config file that exists among
// repoRoot/[FileName] and [UserPath], or "" if neither does. An empty
// repoRoot skips the project file, and an unresolvable home directory
// skips the user file. Stat errors other than "not exist" are returned
// as-is.
func Discover(repoRoot string) (string, error) {
	var candidates []string
	if repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, FileName))
	}
	if userPath, err := UserPath(); err == nil {
		candidates = append(candidates, userPath)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", err
		}
		return path, nil
	}
	return "", nil
}
