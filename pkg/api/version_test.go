package api

import "testing"

func TestShouldNotifyUpdate(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.4.38", "0.4.39", true},
		{"v0.4.38", "0.4.39", true},
		{"0.4.38-next", "0.4.39", true},
		{"0.4.38-12-gdeadbee", "0.4.39", true},
		{"0.4.39", "0.4.39", false},
		{"0.4.39-next", "0.4.39", false},
		{"0.4.39-3-gadfe592", "0.4.39", false},
		{"0.4.40", "0.4.39", false},
		{"0.4.40-next", "0.4.39", false},
		{"CURRENT", "0.4.39", false},
		{"HEAD", "0.4.39", false},
		{"", "0.4.39", false},
		{"adfe592", "0.4.39", false},
		{"0.4.39", "", false},
	}
	for _, tc := range cases {
		got := shouldNotifyUpdate(tc.current, tc.latest)
		if got != tc.want {
			t.Errorf("shouldNotifyUpdate(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestVersionCore(t *testing.T) {
	cases := map[string]string{
		"0.4.39":            "0.4.39",
		"v0.4.39":           "0.4.39",
		"0.4.39-next":       "0.4.39",
		"0.4.39-3-gadfe592": "0.4.39",
		"CURRENT":           "",
		"adfe592":           "",
		"":                  "",
	}
	for in, want := range cases {
		if got := versionCore(in); got != want {
			t.Errorf("versionCore(%q) = %q, want %q", in, got, want)
		}
	}
}
