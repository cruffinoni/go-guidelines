package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
	"github.com/cruffinoni/go-guidelines/internal/config"
	"github.com/cruffinoni/go-guidelines/internal/lint"
)

// scanOptions is the scan configuration after merging the config file with
// flags, flags winning.
type scanOptions struct {
	format   string
	failOn   string
	disabled map[string]bool
}

// resolveOptions loads configPath, if any, and merges it with flags: a
// flag the caller set on cmd wins over the config file, which wins over
// the built-in default already in flags.
func resolveOptions(cat *catalog.Catalog, cmd *cobra.Command, flags scanFlags, configPath string) (scanOptions, error) {
	var file config.File
	if configPath != "" {
		loaded, err := config.Load(configPath)
		if err != nil {
			return scanOptions{}, err
		}
		file = loaded
	}

	format := flags.format
	if !cmd.Flags().Changed("format") && file.Format != nil {
		format = *file.Format
	}
	failOn := flags.failOn
	if !cmd.Flags().Changed("fail-on") && file.FailOn != nil {
		failOn = *file.FailOn
	}

	disabled, err := resolveDisabledRules(cat, file.Rules, flags.enableRules, flags.disableRules)
	if err != nil {
		return scanOptions{}, err
	}
	return scanOptions{format: format, failOn: failOn, disabled: disabled}, nil
}

// resolveDisabledRules merges the config file's rule table with
// --enable-rule and --disable-rule flags, flags winning over the config
// file. Every named rule must exist in cat, and no rule may be named by
// both flags at once.
func resolveDisabledRules(cat *catalog.Catalog, configRules map[string]bool, enable, disable []string) (map[string]bool, error) {
	enabledByFlag := make(map[string]bool, len(enable))
	for _, id := range enable {
		if _, ok := cat.Get(id); !ok {
			return nil, fmt.Errorf("unknown rule %q", id)
		}
		enabledByFlag[id] = true
	}
	disabledByFlag := make(map[string]bool, len(disable))
	for _, id := range disable {
		if _, ok := cat.Get(id); !ok {
			return nil, fmt.Errorf("unknown rule %q", id)
		}
		if enabledByFlag[id] {
			return nil, fmt.Errorf("rule %q is both enabled and disabled", id)
		}
		disabledByFlag[id] = true
	}

	disabled := make(map[string]bool, len(configRules))
	for id, on := range configRules {
		if _, ok := cat.Get(id); !ok {
			return nil, fmt.Errorf("unknown rule %q in config", id)
		}
		if !on {
			disabled[id] = true
		}
	}
	for id := range enabledByFlag {
		delete(disabled, id)
	}
	for id := range disabledByFlag {
		disabled[id] = true
	}
	return disabled, nil
}

// parseFailOn parses the --fail-on value as the minimum severity that
// fails a scan. "none" disables the threshold, reported as the zero
// Severity.
func parseFailOn(s string) (lint.Severity, bool) {
	switch s {
	case "none":
		return 0, true
	case "info":
		return lint.SeverityInfo, true
	case "warning":
		return lint.SeverityWarning, true
	case "error":
		return lint.SeverityError, true
	default:
		return 0, false
	}
}
