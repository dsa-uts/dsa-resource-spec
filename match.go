package resource

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// MatchOutput compares UTF-8 output using the given mode without modifying inputs.
// Exact compares bytes; easy compares whitespace-separated tokens on each line;
// sorted additionally ignores token order within each line, preserving duplicates.
// Easy and sorted recognize LF, CRLF and CR, and remove exactly one final newline.
// Nil slices are empty output. Invalid UTF-8 in either input is a mismatch, even
// in exact mode. An unknown mode (including the empty string) returns an error.
func MatchOutput(actual, expected []byte, mode MatchMode) (bool, error) {
	switch mode {
	case MatchExact, MatchEasy, MatchSorted:
	default:
		return false, fmt.Errorf("invalid output match mode %q", mode)
	}
	if !utf8.Valid(actual) || !utf8.Valid(expected) {
		return false, nil
	}
	if mode == MatchExact {
		return bytes.Equal(actual, expected), nil
	}
	actualLines, expectedLines := outputLines(actual), outputLines(expected)
	if len(actualLines) != len(expectedLines) {
		return false, nil
	}
	for i, line := range actualLines {
		got, want := strings.Fields(line), strings.Fields(expectedLines[i])
		if mode == MatchSorted {
			slices.Sort(got)
			slices.Sort(want)
		}
		if !slices.Equal(got, want) {
			return false, nil
		}
	}
	return true, nil
}

func outputLines(output []byte) []string {
	s := strings.ReplaceAll(string(output), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
