// Package config loads and discovers the gg-lint.toml project
// configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// File is the on-disk shape of gg-lint.toml.
type File struct {
	Format *string         `toml:"format"`
	FailOn *string         `toml:"fail_on"`
	Rules  map[string]bool `toml:"rules"`
}

// Load decodes the TOML file at path into a [File]. Decoding is strict:
// unknown keys are rejected. Every error names path; when go-toml reports a
// position, the error also carries the line and column of the offending
// token.
func Load(path string) (File, error) {
	var file File
	data, err := os.Open(path)
	if err != nil {
		return file, fmt.Errorf("config %s: %w", path, err)
	}
	defer data.Close()

	if err := toml.NewDecoder(data).DisallowUnknownFields().Decode(&file); err != nil {
		if decodeErr, ok := errors.AsType[*toml.DecodeError](err); ok {
			line, column := decodeErr.Position()
			return file, fmt.Errorf("config %s:%d:%d: %w", path, line, column, err)
		}
		return file, fmt.Errorf("config %s: %w", path, err)
	}
	return file, nil
}

// Discover returns the path to root/gg-lint.toml, or "" if no such file
// exists. Stat errors other than "not exist" are returned as-is.
func Discover(root string) (string, error) {
	path := filepath.Join(root, "gg-lint.toml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return path, nil
}
