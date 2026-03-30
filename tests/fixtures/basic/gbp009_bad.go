package util

import "strings"

func EmptySlice() []string {
	s := []string{}
	return s
}

func CopyBuilder(src strings.Builder) strings.Builder {
	var dst strings.Builder = src
	return dst
}
