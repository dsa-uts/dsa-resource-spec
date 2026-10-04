package resource

import (
	"bytes"
	"slices"
	"strings"
)

// MatchOutput compares output using the given mode without modifying inputs.
// Exact compares bytes; easy compares whitespace-separated tokens on each line;
// sorted additionally ignores token order within each line, preserving duplicates.
// Easy and sorted recognize LF, CRLF and CR, and remove exactly one final newline.
// Invalid UTF-8 bytes are preserved and compared.
// An unknown mode (including the empty string) returns false.
func MatchOutput(actual, expected []byte, mode MatchMode) bool {
	switch mode {
	case MatchExact:
		return bytes.Equal(actual, expected)
	case MatchEasy, MatchSorted:
	default:
		return false
	}
	actualLines, expectedLines := outputLines(actual), outputLines(expected)
	if len(actualLines) != len(expectedLines) {
		return false
	}
	for i, line := range actualLines {
		got, want := strings.Fields(line), strings.Fields(expectedLines[i])
		if mode == MatchSorted {
			slices.Sort(got)
			slices.Sort(want)
		}
		if !slices.Equal(got, want) {
			return false
		}
	}
	return true
}

func outputLines(output []byte) []string {
	s := strings.ReplaceAll(string(output), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
