package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMergeSceneFilenames(t *testing.T) {
	cases := []struct {
		name     string
		existing string
		names    []string
		want     []string
		changed  bool
	}{
		{
			name:     "appends new, keeps existing order",
			existing: `["a.mp4","b.mp4"]`,
			names:    []string{"c.mp4"},
			want:     []string{"a.mp4", "b.mp4", "c.mp4"},
			changed:  true,
		},
		{
			name:     "re-append is idempotent",
			existing: `["a.mp4","b.mp4"]`,
			names:    []string{"a.mp4", "b.mp4"},
			want:     []string{"a.mp4", "b.mp4"},
			changed:  false,
		},
		{
			name:     "mixed new and duplicate",
			existing: `["a.mp4"]`,
			names:    []string{"a.mp4", "b.mp4", "a.mp4"},
			want:     []string{"a.mp4", "b.mp4"},
			changed:  true,
		},
		{
			name:     "empty existing builds list",
			existing: ``,
			names:    []string{"a.mp4"},
			want:     []string{"a.mp4"},
			changed:  true,
		},
		{
			name:     "blanks skipped, empty input is no-op",
			existing: `["a.mp4"]`,
			names:    []string{"", "   "},
			want:     []string{"a.mp4"},
			changed:  false,
		},
		{
			name:     "whitespace trimmed before dedupe",
			existing: `["a.mp4"]`,
			names:    []string{"  a.mp4  ", "  b.mp4 "},
			want:     []string{"a.mp4", "b.mp4"},
			changed:  true,
		},
		{
			name:     "match is exact, not normalized",
			existing: `["A.mp4"]`,
			names:    []string{"a.mp4"},
			want:     []string{"A.mp4", "a.mp4"},
			changed:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, err := mergeSceneFilenames(tc.existing, tc.names)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if changed != tc.changed {
				t.Errorf("changed = %v, want %v", changed, tc.changed)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("merged = %q, want %q", got, tc.want)
			}
			// encoding must round-trip through encoding/json exactly,
			// like the scan matcher write path
			enc, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var back []string
			if err := json.Unmarshal(enc, &back); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(back, tc.want) {
				t.Errorf("round-trip = %q, want %q", back, tc.want)
			}
		})
	}
}

func TestMergeSceneFilenamesCorrupt(t *testing.T) {
	if _, _, err := mergeSceneFilenames(`["a.mp4"`, []string{"b.mp4"}); err == nil {
		t.Error("expected error on corrupt existing JSON, got nil")
	}
}

func TestMergeSceneFilenamesEscaping(t *testing.T) {
	// names needing escapes must survive byte-identical to a plain Marshal,
	// i.e. the same bytes the scraper path writes
	tricky := []string{`we"ird.mp4`, `back\slash.mp4`, "unicode-☃.mp4"}
	got, changed, err := mergeSceneFilenames(`[]`, tricky)
	if err != nil || !changed {
		t.Fatalf("err=%v changed=%v", err, changed)
	}
	wantBytes, _ := json.Marshal(tricky)
	gotBytes, _ := json.Marshal(got)
	if string(gotBytes) != string(wantBytes) {
		t.Errorf("encoding differs: %s vs %s", gotBytes, wantBytes)
	}
}
