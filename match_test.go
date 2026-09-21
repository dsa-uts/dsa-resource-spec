package resource_test

import (
	"testing"

	resource "github.com/dsa-uts/dsa-resource-spec"
)

func TestMatchOutput(t *testing.T) {
	for _, tc := range []struct {
		name             string
		actual, expected string
		mode             resource.MatchMode
		match, wantError bool
	}{
		{"exact equal", "日本語\n", "日本語\n", resource.MatchExact, true, false},
		{"exact newline", "A\n", "A", resource.MatchExact, false, false},
		{"exact line endings", "A\r\n", "A\n", resource.MatchExact, false, false},
		{"exact spaces", " A", "A", resource.MatchExact, false, false},
		{"easy whitespace", "  A\tB  \nC　D\n", "A B\nC D", resource.MatchEasy, true, false},
		{"easy line endings", "A\r\nB\rC\r", "A\nB\nC", resource.MatchEasy, true, false},
		{"easy Unicode whitespace", "\u00a0A\u2003B\u2028C\u2029", "A B C", resource.MatchEasy, true, false},
		{"easy token order", "B A", "A B", resource.MatchEasy, false, false},
		{"easy token boundaries", "AB", "A B", resource.MatchEasy, false, false},
		{"easy line boundaries", "A\nB", "A B", resource.MatchEasy, false, false},
		{"easy extra final newline", "A\n\n", "A\n", resource.MatchEasy, false, false},
		{"easy extra CRLF", "A\r\n\r\n", "A", resource.MatchEasy, false, false},
		{"easy interior blank line", "A\n\nB", "A\nB", resource.MatchEasy, false, false},
		{"easy leading blank line", "\nA", "A", resource.MatchEasy, false, false},
		{"easy empty line", "\n", "", resource.MatchEasy, true, false},
		{"easy blank line", " \t\n", "", resource.MatchEasy, true, false},
		{"easy case", "a", "A", resource.MatchEasy, false, false},
		{"sorted tokens", "B A\n3 2 1", "A B\n1 2 3", resource.MatchSorted, true, false},
		{"sorted whitespace", " B\tA\r\n3　2 1\r", "A B\n1 2 3", resource.MatchSorted, true, false},
		{"sorted line order", "B\nA", "A\nB", resource.MatchSorted, false, false},
		{"sorted duplicates", "A B A", "B A A", resource.MatchSorted, true, false},
		{"sorted duplicate count", "A A B", "A B B", resource.MatchSorted, false, false},
		{"sorted extra newline", "B A\n\n", "A B", resource.MatchSorted, false, false},
		{"sorted no numeric conversion", "01 2", "1 2", resource.MatchSorted, false, false},
		{"empty mode", "A", "A", "", false, true},
		{"unknown mode", "A", "A", "unknown", false, true},
		{"mode checked before UTF-8", "\xff", "\xff", "unknown", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, expected := []byte(tc.actual), []byte(tc.expected)
			got, err := resource.MatchOutput(actual, expected, tc.mode)
			if got != tc.match || (err != nil) != tc.wantError {
				t.Fatalf("MatchOutput = (%v, %v), want (%v, error=%v)", got, err, tc.match, tc.wantError)
			}
			if string(actual) != tc.actual || string(expected) != tc.expected {
				t.Fatal("inputs modified")
			}
		})
	}
	for _, mode := range []resource.MatchMode{resource.MatchExact, resource.MatchEasy, resource.MatchSorted} {
		t.Run(string(mode), func(t *testing.T) {
			for _, tc := range []struct {
				actual, expected []byte
				match            bool
			}{
				{nil, []byte{}, true}, {nil, nil, true},
				{[]byte{0xff}, []byte{0xff}, false},
				{[]byte{0xff}, []byte("A"), false},
				{[]byte("A"), []byte{0xff}, false},
				{[]byte("\uFFFD"), []byte("\uFFFD"), true},
			} {
				got, err := resource.MatchOutput(tc.actual, tc.expected, mode)
				if got != tc.match || err != nil {
					t.Fatalf("MatchOutput(%q, %q) = (%v, %v), want (%v, nil)", tc.actual, tc.expected, got, err, tc.match)
				}
			}
		})
	}
}
