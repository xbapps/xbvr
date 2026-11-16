package llmscrape

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestExtractionSchemaShape(t *testing.T) {
	s := ExtractionSchema(3, 1, 2)
	props := s["properties"].(map[string]any)
	required := s["required"].([]string)
	if len(required) != len(props) {
		t.Fatalf("strict mode needs every property required: %d required, %d properties", len(required), len(props))
	}
	for _, r := range required {
		if _, ok := props[r]; !ok {
			t.Errorf("required %q is not a property", r)
		}
	}
	if s["additionalProperties"] != false {
		t.Error("additionalProperties must be false")
	}

	cover := props["cover_image"].(map[string]any)["enum"].([]any)
	if len(cover) != 4 || cover[0] != -1 || cover[3] != 2 {
		t.Errorf("cover enum = %v, want [-1 0 1 2]", cover)
	}
	trailer := props["trailer"].(map[string]any)["enum"].([]any)
	if len(trailer) != 2 {
		t.Errorf("trailer enum = %v, want [-1 0]", trailer)
	}
	follow := props["follow_link"].(map[string]any)["enum"].([]any)
	if len(follow) != 3 || follow[0] != -1 || follow[2] != 1 {
		t.Errorf("follow_link enum = %v, want [-1 0 1]", follow)
	}

	// The schema has to survive JSON encoding to be sent to the server.
	if _, err := json.Marshal(s); err != nil {
		t.Fatal(err)
	}
}

func TestExtractionSchemaNoCandidates(t *testing.T) {
	s := ExtractionSchema(0, 0, 0)
	props := s["properties"].(map[string]any)
	if e := props["cover_image"].(map[string]any)["enum"].([]any); len(e) != 1 || e[0] != -1 {
		t.Errorf("with no images the only cover choice is -1, got %v", e)
	}
	if props["gallery_images"].(map[string]any)["maxItems"] != 0 {
		t.Error("with no images the gallery must be empty")
	}
	if e := props["follow_link"].(map[string]any)["enum"].([]any); len(e) != 1 || e[0] != -1 {
		t.Errorf("with no links the only follow choice is -1, got %v", e)
	}
}

func TestParseExtractionRepairs(t *testing.T) {
	page := &Page{
		Images: []Candidate{{URL: "a"}, {URL: "b"}, {URL: "c"}},
		Videos: []Candidate{{URL: "v"}},
	}
	raw := json.RawMessage(`{
		"is_scene_page": true, "page_kind": "made_up_kind", "match_confidence": 1.7,
		"reason": "  looks   right  ", "title": " Title ", "studio": "S", "site": "", "site_scene_id": "",
		"cast": ["Jane Doe", " jane doe ", "", "Ann Lee"], "tags": ["VR", "vr", "POV"],
		"synopsis": "  text ", "released": "2025-02-30", "duration_minutes": 9999,
		"cover_image": 7, "gallery_images": [1, 1, -3, 2, 99, 0], "trailer": 4
	}`)
	e, err := ParseExtraction(raw, page)
	if err != nil {
		t.Fatal(err)
	}
	if e.PageKind != KindOther {
		t.Errorf("unknown page kind should become %q, got %q", KindOther, e.PageKind)
	}
	if e.MatchConfidence != 1 {
		t.Errorf("confidence should clamp to 1, got %v", e.MatchConfidence)
	}
	if e.Released != "" {
		t.Errorf("impossible date should be cleared, got %q", e.Released)
	}
	if e.DurationMinutes != 0 {
		t.Errorf("absurd duration should be cleared, got %d", e.DurationMinutes)
	}
	if e.CoverImage != -1 || e.Trailer != -1 {
		t.Errorf("out of range indexes should become -1: cover %d trailer %d", e.CoverImage, e.Trailer)
	}
	if got := e.GalleryImages; len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 0 {
		t.Errorf("gallery = %v, want [1 2 0]", got)
	}
	if len(e.Cast) != 2 || e.Cast[0] != "Jane Doe" || e.Cast[1] != "Ann Lee" {
		t.Errorf("cast = %v", e.Cast)
	}
	if len(e.Tags) != 2 {
		t.Errorf("tags should dedupe case-insensitively, got %v", e.Tags)
	}
	if e.Title != "Title" || e.Reason != "looks right" || e.Synopsis != "text" {
		t.Errorf("whitespace not cleaned: %q %q %q", e.Title, e.Reason, e.Synopsis)
	}
}

func TestParseExtractionCoverNotRepeatedInGallery(t *testing.T) {
	page := &Page{Images: []Candidate{{URL: "a"}, {URL: "b"}}}
	raw := json.RawMessage(`{"is_scene_page":true,"page_kind":"official_studio","match_confidence":0.9,"reason":"",
		"title":"","studio":"","site":"","site_scene_id":"","cast":[],"tags":[],"synopsis":"","released":"",
		"duration_minutes":0,"cover_image":0,"gallery_images":[0,1],"trailer":-1}`)
	e, err := ParseExtraction(raw, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.GalleryImages) != 1 || e.GalleryImages[0] != 1 {
		t.Errorf("cover should not also be in the gallery, got %v", e.GalleryImages)
	}
}

func TestParseExtractionRejectsWrongShape(t *testing.T) {
	if _, err := ParseExtraction(json.RawMessage(`{"cast": "not a list"}`), &Page{}); err == nil {
		t.Error("expected an error for a mistyped field")
	}
}

func TestBuildMessagesListsCandidatesByIndex(t *testing.T) {
	page := &Page{URL: "https://x/s", Title: "T", Meta: map[string]string{"og:title": "T"},
		Images: []Candidate{{URL: "https://x/a.jpg", Hint: "og:image"}}, Text: "body"}
	msgs := BuildMessages(page, &FileInfo{Filename: "x.mp4", DurationMinutes: 42, Context: "studio is X"})
	if len(msgs) != 2 || msgs[0].Role != "system" {
		t.Fatalf("messages = %+v", msgs)
	}
	u := msgs[1].Content
	for _, want := range []string{"[0] https://x/a.jpg (og:image)", "file duration: 42 minutes", "notes from the library owner: studio is X", "VIDEO CANDIDATES\n(none)"} {
		if !strings.Contains(u, want) {
			t.Errorf("prompt missing %q:\n%s", want, u)
		}
	}
}

func TestFieldGuideReachesThePrompt(t *testing.T) {
	msgs := BuildMessages(&Page{URL: "https://x", Meta: map[string]string{}}, nil)
	sys := msgs[0].Content
	// Every described field must be in the system prompt, in schema order.
	for _, want := range []string{"- is_scene_page:", "- page_kind:", "download_or_piracy: file hosts", "- title:", "'Jane Doe: Summer Heat'", "- trailer:"} {
		if !strings.Contains(sys, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
	if strings.Index(sys, "- is_scene_page:") > strings.Index(sys, "- trailer:") {
		t.Error("field guide should follow schema order")
	}
}

func TestStripBoilerplate(t *testing.T) {
	cases := map[string]string{
		"Cleo Vega: Sweet Pink Bunny VR porn video featuring Cleo Vega from Ethernal VR. Currently available for online streaming and download in 4K-8K virtual reality here on VRPorn.com.": "",
		"Lena is a gamer with a teasing spark. She bets you can't win. Watch it now on ExampleVR.com!":                                                                                       "Lena is a gamer with a teasing spark. She bets you can't win.",
		"A quiet evening turns intense. Is she ready?": "A quiet evening turns intense. Is she ready?",
		"": "",
	}
	for in, want := range cases {
		if got := StripBoilerplate(in); got != want {
			t.Errorf("StripBoilerplate(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestApplyDurationCheck(t *testing.T) {
	file := &FileInfo{DurationMinutes: 41}
	pmv := &Extraction{MatchConfidence: 0.9, DurationMinutes: 13, Reason: "Title matches."}
	ApplyDurationCheck(pmv, file)
	if pmv.MatchConfidence != 0.3 || !strings.Contains(pmv.Reason, "13 min does not match the file's 41 min") {
		t.Errorf("a 13 min page against a 41 min file should be capped: %v %q", pmv.MatchConfidence, pmv.Reason)
	}

	close := &Extraction{MatchConfidence: 0.95, DurationMinutes: 40}
	ApplyDurationCheck(close, file)
	if close.MatchConfidence != 0.95 {
		t.Errorf("40 vs 41 min is within tolerance, confidence changed to %v", close.MatchConfidence)
	}

	short := &Extraction{MatchConfidence: 0.9, DurationMinutes: 12}
	ApplyDurationCheck(short, &FileInfo{DurationMinutes: 10})
	if short.MatchConfidence != 0.9 {
		t.Error("short files get a 3 minute floor on the tolerance")
	}

	unknown := &Extraction{MatchConfidence: 0.9, DurationMinutes: 0}
	ApplyDurationCheck(unknown, file)
	ApplyDurationCheck(pmv, nil)
	if unknown.MatchConfidence != 0.9 {
		t.Error("an unknown page duration must not change confidence")
	}
}

func TestApplyNameCheck(t *testing.T) {
	cases := []struct {
		name   string
		file   *FileInfo
		title  string
		cast   []string
		capped bool
	}{
		{"one shared word is not a match", &FileInfo{Filename: "creampie-me-please-pt2.mp4"},
			"Cleo Gets a Massage With Creampie Included", []string{"Cleo Vega"}, true},
		{"title in the filename", &FileInfo{Filename: "cleo-vega-sweet-pink-bunny.mp4"},
			"Sweet Pink Bunny", []string{"Cleo Vega"}, false},
		{"performers run together in the filename", &FileInfo{Filename: "FPVR-AliciaWilliams-DannySteele-180-POV_8K_UHD.mp4"},
			"VR Sneak & Swap", []string{"Alicia Williams"}, false},
		{"title only in the scene's own folder", &FileInfo{Filename: "vac-ddfnvr180109kqsdq-2160.mp4",
			Folder: "DDFNetworkVR.18.01.09.Kira.Queen.Sugar.Daddy.for.the.Queen.XXX.VR180.2160p.MP4-VACCiNE", FolderIsScene: true},
			"Sugar Daddy for the Queen", nil, false},
		{"the same folder shared with other videos says nothing", &FileInfo{Filename: "vac-ddfnvr180109kqsdq-2160.mp4",
			Folder: "DDFNetworkVR.18.01.09.Kira.Queen.Sugar.Daddy.for.the.Queen.XXX.VR180.2160p.MP4-VACCiNE"},
			"Sugar Daddy for the Queen", nil, true},
		{"user notes count", &FileInfo{Filename: "clip01.mp4", Context: "Lena from Slutty House"},
			"Slutty House", nil, false},
	}
	for _, c := range cases {
		e := &Extraction{MatchConfidence: 0.9, Title: c.title, Cast: c.cast}
		ApplyNameCheck(e, c.file)
		if capped := e.MatchConfidence == 0.4; capped != c.capped {
			t.Errorf("%s: confidence %.2f, capped=%v want %v (%s)", c.name, e.MatchConfidence, capped, c.capped, e.Reason)
		}
	}

	nothing := &Extraction{MatchConfidence: 0.9, Title: "Anything"}
	ApplyNameCheck(nothing, &FileInfo{Filename: "vr4_2x.mp4"})
	if nothing.MatchConfidence != 0.9 {
		t.Error("with no words to compare, confidence must be left alone")
	}
}

func TestDurationPreviewFromTheSameStudio(t *testing.T) {
	file := &FileInfo{Filename: "vac-ddfnvr180109kqsdq-2160.mp4", DurationMinutes: 27, FolderIsScene: true,
		Folder: "DDFNetworkVR.18.01.09.Kira.Queen.Sugar.Daddy.for.the.Queen.XXX.VR180.2160p.MP4-VACCiNE"}
	tube := &Extraction{MatchConfidence: 0.9, DurationMinutes: 12, Studio: "DDF Network VR", Title: "Sugar Daddy for the Queen"}
	ApplyDurationCheck(tube, file)
	if tube.MatchConfidence != 0.9 || tube.DurationMinutes != 0 || !strings.Contains(tube.Reason, "preview") {
		t.Errorf("a same-studio page with a short clip is a preview: conf %.2f dur %d %q", tube.MatchConfidence, tube.DurationMinutes, tube.Reason)
	}

	edit := &Extraction{MatchConfidence: 0.9, DurationMinutes: 12, Studio: "HeatFlow PMV", Title: "Sugar Daddy for the Queen"}
	ApplyDurationCheck(edit, file)
	if edit.MatchConfidence != 0.3 {
		t.Errorf("another studio's short version is an edit and should be capped, got %.2f", edit.MatchConfidence)
	}

	game := &Extraction{MatchConfidence: 0.9, DurationMinutes: 180, IsParentPage: true}
	ApplyDurationCheck(game, file)
	if game.MatchConfidence != 0.9 || game.DurationMinutes != 0 {
		t.Errorf("a game page's total running time must be dropped, not held against the part: %.2f %d", game.MatchConfidence, game.DurationMinutes)
	}
}

func TestParentPageFields(t *testing.T) {
	raw := func(scene, parent bool, part string) json.RawMessage {
		b, _ := json.Marshal(map[string]any{"is_scene_page": scene, "is_parent_page": parent, "part_title": part,
			"page_kind": "official_studio", "match_confidence": 0.9, "reason": "", "title": "Seductive Science",
			"studio": "", "site": "", "site_scene_id": "", "cast": []string{}, "tags": []string{}, "synopsis": "",
			"released": "", "duration_minutes": 0, "cover_image": -1, "gallery_images": []int{}, "trailer": -1})
		return b
	}
	e, _ := ParseExtraction(raw(false, true, " Ember  Moans "), &Page{})
	if !e.IsParentPage || e.PartTitle != "Ember Moans" {
		t.Errorf("parent page kept wrongly: %v %q", e.IsParentPage, e.PartTitle)
	}
	e, _ = ParseExtraction(raw(true, true, "Ember Moans"), &Page{})
	if e.IsParentPage || e.PartTitle != "" {
		t.Error("a page about the part itself wins over a parent page")
	}
	e, _ = ParseExtraction(raw(false, false, "stray"), &Page{})
	if e.PartTitle != "" {
		t.Error("part_title only means something on a parent page")
	}
}

func TestExtractionSchemaCapsEveryStringAndList(t *testing.T) {
	props := ExtractionSchema(3, 1, 2)["properties"].(map[string]any)
	for name, v := range props {
		p := v.(map[string]any)
		switch p["type"] {
		case "string":
			if _, ok := p["maxLength"]; !ok && p["pattern"] == nil && p["enum"] == nil {
				t.Errorf("%s: string without maxLength can run the answer past its token budget", name)
			}
		case "array":
			if _, ok := p["maxItems"]; !ok {
				t.Errorf("%s: list without maxItems", name)
			}
		}
	}
}

func TestParseExtractionFollowLink(t *testing.T) {
	page := &Page{Links: []Candidate{{URL: "https://example.com/one"}, {URL: "https://example.com/two"}}}
	raw := func(scene, parent bool, follow int) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{"is_scene_page": %v, "is_parent_page": %v, "follow_link": %d}`, scene, parent, follow))
	}
	cases := []struct {
		name   string
		scene  bool
		parent bool
		follow int
		want   int
	}{
		{"overview page points at the scene", false, false, 1, 1},
		{"no link chosen", false, false, -1, -1},
		{"link index out of range", false, false, 9, -1},
		{"negative index", false, false, -4, -1},
		{"scene page never follows", true, false, 0, -1},
		{"parent page never follows", false, true, 0, -1},
	}
	for _, c := range cases {
		e, err := ParseExtraction(raw(c.scene, c.parent, c.follow), page)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if e.FollowLink != c.want {
			t.Errorf("%s: FollowLink = %d, want %d", c.name, e.FollowLink, c.want)
		}
	}
}
