package gbp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgumentVerbs(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   map[int]rune
		ok     bool
	}{
		{name: "no verbs", format: "plain text", want: map[int]rune{}, ok: true},
		{name: "escaped percent", format: "100%% done", want: map[int]rune{}, ok: true},
		{name: "single verb", format: "%s", want: map[int]rune{1: 's'}, ok: true},
		{name: "wrap verb", format: "%w", want: map[int]rune{1: 'w'}, ok: true},
		{name: "wrap before value", format: "%w and %v", want: map[int]rune{1: 'w', 2: 'v'}, ok: true},
		{name: "value before wrap", format: "%v and %w", want: map[int]rune{1: 'v', 2: 'w'}, ok: true},
		{name: "explicit index advances position", format: "%[2]w %v", want: map[int]rune{2: 'w', 3: 'v'}, ok: true},
		{name: "escaped percent hides literal w", format: "progress 100%%w: %v", want: map[int]rune{1: 'v'}, ok: true},
		{name: "star width consumes an argument", format: "%*d %s", want: map[int]rune{1: '*', 2: 'd', 3: 's'}, ok: true},
		{name: "star precision consumes an argument", format: "%.*f %w", want: map[int]rune{1: '*', 2: 'f', 3: 'w'}, ok: true},
		{name: "indexed width and precision", format: "%[3]*.[2]*[1]f", want: map[int]rune{3: '*', 2: '*', 1: 'f'}, ok: true},
		{name: "flags", format: "%+q", want: map[int]rune{1: 'q'}, ok: true},
		{name: "dangling percent", format: "100%", ok: false},
		{name: "zero index", format: "%[0]s", ok: false},
		{name: "unterminated index", format: "%[1s", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := argumentVerbs(tt.format)
			assert.Equal(t, tt.ok, ok, "format %q", tt.format)
			if !ok {
				return
			}
			assert.Equal(t, tt.want, got, "format %q", tt.format)
		})
	}
}
