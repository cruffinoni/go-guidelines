package gbp

import "strings"

// argumentVerbs maps each fmt argument position, numbered from 1, to the verb
// that consumes it. A position consumed by a "*" width or precision is recorded
// with '*' because a number can never carry an error chain.
//
// ok is false when format cannot be mapped reliably, such as a dangling "%" or a
// zero argument index. Callers must then fail closed rather than guess.
func argumentVerbs(format string) (verbs map[int]rune, ok bool) {
	verbs = make(map[int]rune)
	// next is the position the next verb or star consumes.
	next := 1
	for i := 0; i < len(format); {
		if format[i] != '%' {
			i++
			continue
		}
		i++
		if i < len(format) && format[i] == '%' {
			i++ // "%%" is a literal percent and consumes no argument.
			continue
		}
		// An explicit [n] index may precede the flags, the width, the precision,
		// and the verb; each one moves the current position to n.
		readIndex := func() bool {
			if i >= len(format) || format[i] != '[' {
				return true
			}
			j := i + 1
			n := 0
			for j < len(format) && format[j] >= '0' && format[j] <= '9' {
				n = n*10 + int(format[j]-'0')
				j++
			}
			if j >= len(format) || format[j] != ']' || n == 0 {
				return false
			}
			next = n
			i = j + 1
			return true
		}
		if !readIndex() {
			return nil, false
		}
		for i < len(format) && strings.ContainsRune("+-# 0", rune(format[i])) {
			i++
		}
		if !readIndex() {
			return nil, false
		}
		if i < len(format) && format[i] == '*' {
			verbs[next] = '*'
			next++
			i++
		} else {
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				i++
			}
		}
		if !readIndex() {
			return nil, false
		}
		if i < len(format) && format[i] == '.' {
			i++
			if i < len(format) && format[i] == '*' {
				verbs[next] = '*'
				next++
				i++
			} else {
				for i < len(format) && format[i] >= '0' && format[i] <= '9' {
					i++
				}
			}
		}
		if !readIndex() {
			return nil, false
		}
		if i >= len(format) {
			return nil, false // a "%" with no verb.
		}
		verbs[next] = rune(format[i])
		next++
		i++
	}
	return verbs, true
}
