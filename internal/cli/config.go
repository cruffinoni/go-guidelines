package cli

import (
	"bufio"
	"fmt"
	"github.com/cruffinoni/go-guidelines/internal/rules"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type scanConfig struct {
	format string
	failOn string
	rules  map[string]bool
}

func defaultConfig() scanConfig {
	ruleSettings := make(map[string]bool)
	for _, rule := range rules.BuiltInRules() {
		ruleSettings[rule.ID()] = rule.ID() == "GBP001"
	}
	return scanConfig{format: "text", failOn: "error", rules: ruleSettings}
}

func discoverProjectConfig() (string, error) {
	output, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", nil
	}
	root := strings.TrimSpace(string(output))
	path := filepath.Join(root, "gg-lint.toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return path, nil
}

func loadConfig(path string, rulesByID map[string]struct{}) (scanConfig, error) {
	config := defaultConfig()
	if path == "" {
		return config, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return config, fmt.Errorf("read config %s: %w", path, err)
	}
	defer file.Close()

	section := ""
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(stripTOMLComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "rules" {
				return config, configLineError(path, lineNumber, "unsupported section "+strconv.Quote(section))
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return config, configLineError(path, lineNumber, "expected key = value")
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		fullKey := key
		if section != "" {
			fullKey = section + "." + key
		}
		if seen[fullKey] {
			return config, configLineError(path, lineNumber, "duplicate key "+strconv.Quote(fullKey))
		}
		seen[fullKey] = true
		if section == "rules" {
			if _, exists := rulesByID[key]; !exists {
				return config, configLineError(path, lineNumber, "unknown rule "+strconv.Quote(key))
			}
			parsed, parseErr := strconv.ParseBool(value)
			if parseErr != nil {
				return config, configLineError(path, lineNumber, "rule value must be true or false")
			}
			config.rules[key] = parsed
			continue
		}
		switch key {
		case "format", "fail_on":
			parsed, parseErr := strconv.Unquote(value)
			if parseErr != nil {
				return config, configLineError(path, lineNumber, key+" must be a quoted string")
			}
			if key == "format" {
				config.format = parsed
			} else {
				config.failOn = parsed
			}
		default:
			return config, configLineError(path, lineNumber, "unknown key "+strconv.Quote(key))
		}
	}
	if err := scanner.Err(); err != nil {
		return config, fmt.Errorf("read config %s: %w", path, err)
	}
	if config.format != "text" && config.format != "json" {
		return config, fmt.Errorf("config %s: format must be text or json", path)
	}
	if _, ok := severityRank(config.failOn); !ok {
		return config, fmt.Errorf("config %s: invalid fail_on value %q", path, config.failOn)
	}
	return config, nil
}

func configLineError(path string, line int, message string) error {
	return fmt.Errorf("config %s:%d: %s", path, line, message)
}

func stripTOMLComment(line string) string {
	quoted, escaped := false, false
	for index, char := range line {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quoted {
			escaped = true
			continue
		}
		if char == '"' {
			quoted = !quoted
		}
		if char == '#' && !quoted {
			return line[:index]
		}
	}
	return line
}
