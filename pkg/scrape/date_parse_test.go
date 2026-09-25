package scrape

import "testing"

func TestParseGroobyDate(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"September 11, 2025", "2025-09-11"},
		{"Added September 11, 2025", "2025-09-11"},
		{"  Added September 1, 2025  ", "2025-09-01"},
		{"", ""},
		{"AB", ""},             // 2-char page drift, the #2279 shape
		{"Added ", ""},         // prefix with no date behind it
		{"Coming soon", ""},    // free text, not a date
		{"11/09/2025", ""},     // wrong shape entirely
		{"September 2025", ""}, // missing day
	} {
		if got := parseGroobyDate(tc.in); got != tc.want {
			t.Errorf("parseGroobyDate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
