package cli

import (
	"strings"

	"github.com/cruffinoni/go-guidelines/internal/rules"
)

func isFlagProvided(args []string, name string) bool {
	long, short := "--"+name, "-"+name
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == long || arg == short || strings.HasPrefix(arg, long+"=") || strings.HasPrefix(arg, short+"=") {
			return true
		}
	}
	return false
}

type stringList []string

// String returns the registered values as a comma-separated string.
func (s *stringList) String() string { return strings.Join(*s, ",") }

// Set appends one command-line value to the list.
func (s *stringList) Set(value string) error { *s = append(*s, value); return nil }
func severityRank(value string) (int, bool) {
	rank, ok := map[string]int{"none": -1, "info": 1, "warning": 2, "error": 3}[value]
	return rank, ok
}
func severityRankValue(value rules.Severity) int { rank, _ := severityRank(string(value)); return rank }
