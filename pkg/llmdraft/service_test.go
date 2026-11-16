package llmdraft

// These tests import XBVR's models package, whose init opens a database in the app directory.
// Point that somewhere disposable:
//
//	XBVR_APPDIR=$(mktemp -d) go test ./pkg/llmdraft/

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xbapps/xbvr/pkg/llmscrape"
	"github.com/xbapps/xbvr/pkg/models"
)

func TestBuildScrapedSceneWithoutCoverLeavesCoversNil(t *testing.T) {
	page := &llmscrape.Page{URL: "https://examplevr.com/scene/9", Title: "llmscrape.Page title"}
	ext := &llmscrape.Extraction{Site: "ExampleVR", CoverImage: -1, Trailer: -1}
	scene := BuildScrapedScene(page, ext, nil)

	// The scene import reads Covers[0] whenever Covers is non-nil, so an empty slice would panic.
	// The draft is stored as JSON, so the round trip must preserve nil too.
	var d models.DraftScene
	if err := d.SetScene(scene); err != nil {
		t.Fatal(err)
	}
	back, err := d.Scene()
	if err != nil {
		t.Fatal(err)
	}
	if back.Covers != nil {
		t.Fatalf("Covers = %#v after a JSON round trip; must be nil when there is no cover", back.Covers)
	}
	if scene.Title != "llmscrape.Page title" {
		t.Errorf("empty extracted title should fall back to the page title, got %q", scene.Title)
	}
}

func TestBuildScrapedScene(t *testing.T) {
	page := &llmscrape.Page{
		URL:          "https://examplevr.com/scene/1234?ref=x",
		CanonicalURL: "https://examplevr.com/scene/1234",
		Images:       []llmscrape.Candidate{{URL: "https://c/cover.jpg"}, {URL: "https://c/s1.jpg"}, {URL: "https://c/s2.jpg"}},
		Videos:       []llmscrape.Candidate{{URL: "https://c/trailer.mp4"}},
	}
	ext := &llmscrape.Extraction{
		Title: "Sweet Pink Bunny", Studio: "Ethernal VR", Site: "", SiteSceneID: "1234",
		Cast: []string{"Cleo Vega"}, Tags: []string{"POV"}, Released: "2025-03-03", DurationMinutes: 40,
		CoverImage: 0, GalleryImages: []int{2, 1}, Trailer: 0,
	}
	s := BuildScrapedScene(page, ext, &llmscrape.FileInfo{Filename: "cleo-vega-sweet-pink-bunny.mp4"})

	if s.SceneID != "llm-ethernal-vr-1234" {
		t.Errorf("SceneID = %q", s.SceneID)
	}
	if s.Site != "Ethernal VR" || s.Studio != "Ethernal VR" {
		t.Errorf("site/studio = %q/%q; site should fall back to the studio", s.Site, s.Studio)
	}
	if s.HomepageURL != "https://examplevr.com/scene/1234" {
		t.Errorf("homepage should prefer the canonical URL, got %q", s.HomepageURL)
	}
	if len(s.Covers) != 1 || s.Covers[0] != "https://c/cover.jpg" {
		t.Errorf("covers = %v", s.Covers)
	}
	if strings.Join(s.Gallery, ",") != "https://c/s2.jpg,https://c/s1.jpg" {
		t.Errorf("gallery should keep the model's order, got %v", s.Gallery)
	}
	if s.TrailerType != "url" || s.TrailerSrc != "https://c/trailer.mp4" {
		t.Errorf("trailer = %q %q", s.TrailerType, s.TrailerSrc)
	}
	if len(s.Filenames) != 1 || s.Filenames[0] != "cleo-vega-sweet-pink-bunny.mp4" {
		t.Errorf("filenames = %v", s.Filenames)
	}
	if s.Duration != 40 || s.Released != "2025-03-03" || s.SceneType != "VR" {
		t.Errorf("duration/released/type = %d %q %q", s.Duration, s.Released, s.SceneType)
	}
	if _, err := json.Marshal(s); err != nil {
		t.Fatal(err)
	}
}

func TestSceneIDIsStableWithoutSiteID(t *testing.T) {
	page := &llmscrape.Page{URL: "https://examplevr.com/a", CanonicalURL: "https://examplevr.com/scene/abc"}
	a := BuildScrapedScene(page, &llmscrape.Extraction{Site: "ExampleVR", CoverImage: -1, Trailer: -1}, nil)
	b := BuildScrapedScene(page, &llmscrape.Extraction{Site: "ExampleVR", CoverImage: -1, Trailer: -1}, nil)
	if a.SceneID != b.SceneID {
		t.Errorf("scraping the same page twice gave %q and %q", a.SceneID, b.SceneID)
	}
	if !strings.HasPrefix(a.SceneID, "llm-examplevr-") {
		t.Errorf("SceneID = %q", a.SceneID)
	}
	other := &llmscrape.Page{URL: "https://examplevr.com/scene/xyz"}
	c := BuildScrapedScene(other, &llmscrape.Extraction{Site: "ExampleVR", CoverImage: -1, Trailer: -1}, nil)
	if c.SceneID == a.SceneID {
		t.Error("different pages must not share a SceneID")
	}
}

func TestDomainBlocked(t *testing.T) {
	domains := ParseDomains("hqcollect.is, www.vrpornx.net\nexample.org")
	if len(domains) != 3 || domains[1] != "vrpornx.net" {
		t.Fatalf("ParseDomains = %v", domains)
	}
	cases := map[string]bool{
		"https://hqcollect.is/siterip/1":     true,
		"https://www.vrpornx.net/x":          true,
		"https://cdn.example.org/y":          true,
		"https://notexample.org/z":           false,
		"https://dezyred.com/games/slutty":   false,
		"https://hqcollect.is.evil.com/path": false,
	}
	for u, want := range cases {
		if got := DomainBlocked(u, domains); got != want {
			t.Errorf("DomainBlocked(%s) = %v, want %v", u, got, want)
		}
	}
}

func TestSiteComesFromStudioOnAggregatorPages(t *testing.T) {
	page := &llmscrape.Page{URL: "https://vrporn.com/cleo-vega-sweet-pink-bunny/"}
	agg := BuildScrapedScene(page, &llmscrape.Extraction{PageKind: llmscrape.KindStore, Site: "VRPorn.com",
		Studio: "Ethernal VR", CoverImage: -1, Trailer: -1}, nil)
	if agg.Site != "Ethernal VR" {
		t.Errorf("aggregator page: site = %q, want the studio", agg.Site)
	}

	own := BuildScrapedScene(page, &llmscrape.Extraction{PageKind: llmscrape.KindOfficial, Site: "SLR Originals",
		Studio: "SexLikeReal", CoverImage: -1, Trailer: -1}, nil)
	if own.Site != "SLR Originals" || own.Studio != "SexLikeReal" {
		t.Errorf("official page: site/studio = %q/%q; the page's own site name should be kept", own.Site, own.Studio)
	}
}

func TestBrandName(t *testing.T) {
	cases := map[string]string{"Dezyred.com": "Dezyred", "VRPorn.com": "VRPorn", "Ethernal VR": "Ethernal VR",
		"cdn.example.com": "cdn.example.com", " RealJamVR ": "RealJamVR", "VR-Bangers.net": "VR-Bangers"}
	for in, want := range cases {
		if got := brandName(in); got != want {
			t.Errorf("brandName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRetryIfBusy(t *testing.T) {
	calls := 0
	err := models.RetryIfBusy(func() error {
		calls++
		if calls < 3 {
			return errors.New("database is locked")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("lock errors should be retried until success: err=%v calls=%d", err, calls)
	}

	calls = 0
	err = models.RetryIfBusy(func() error {
		calls++
		return errors.New("UNIQUE constraint failed")
	})
	if err == nil || calls != 1 {
		t.Errorf("other errors must not be retried: err=%v calls=%d", err, calls)
	}
}

func TestCoverFallsBackToFirstStill(t *testing.T) {
	page := &llmscrape.Page{URL: "https://x/s", Images: []llmscrape.Candidate{{URL: "a"}, {URL: "b"}, {URL: "c"}}}
	s := BuildScrapedScene(page, &llmscrape.Extraction{Site: "X", CoverImage: -1, GalleryImages: []int{2, 0}, Trailer: -1}, nil)
	if len(s.Covers) != 1 || s.Covers[0] != "c" || len(s.Gallery) != 1 || s.Gallery[0] != "a" {
		t.Errorf("covers %v gallery %v; the first still should become the cover", s.Covers, s.Gallery)
	}
}

func draftFor(t *testing.T, site, title string, conf float64, kind string, covers []string) *models.DraftScene {
	d := &models.DraftScene{MatchConfidence: conf, PageKind: kind, Status: models.DraftStatusDraft}
	if err := d.SetScene(models.ScrapedScene{Site: site, Title: title, Covers: covers}); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCollapseDuplicatesKeepsTheBest(t *testing.T) {
	results := []ScrapeResult{
		{URL: "u1", Draft: draftFor(t, "Ethernal VR", "Jedi Mind Tricks… of Pleasure", 1, llmscrape.KindStore, nil)},
		{URL: "u2", Draft: draftFor(t, "Ethernal VR", "Jedi Mind Tricks... of Pleasure", 1, llmscrape.KindStore, []string{"cover"})},
		{URL: "u3", Draft: draftFor(t, "ethernal vr", "JEDI MIND TRICKS OF PLEASURE", 0.9, llmscrape.KindOfficial, []string{"c"})},
		{URL: "u4", Draft: draftFor(t, "Other", "Different Scene", 0.2, llmscrape.KindStore, nil)},
		{URL: "u5", Skipped: "not a scene page"},
	}
	collapseDuplicates(results)

	if results[1].Draft.Status != models.DraftStatusDraft {
		t.Errorf("u2 (same confidence, has a cover) should be kept, status %q", results[1].Draft.Status)
	}
	for _, i := range []int{0, 2} {
		if results[i].Draft.Status != models.DraftStatusDismissed || !strings.Contains(results[i].Skipped, "u2") {
			t.Errorf("%s should be dismissed as a duplicate of u2: %q %q", results[i].URL, results[i].Draft.Status, results[i].Skipped)
		}
	}
	if results[3].Draft.Status != models.DraftStatusDraft {
		t.Error("a different scene must not be collapsed")
	}
}

func TestParentPageGivesEachPartItsOwnScene(t *testing.T) {
	page := &llmscrape.Page{URL: "https://dezyred.com/games/seductive-science"}
	part := func(name string) models.ScrapedScene {
		return BuildScrapedScene(page, &llmscrape.Extraction{PageKind: llmscrape.KindOfficial, IsParentPage: true,
			PartTitle: name, Title: "Seductive Science", Studio: "Dezyred", CoverImage: -1, Trailer: -1}, nil)
	}
	a, b := part("Ember Moans"), part("Final Exam")
	if a.Title != "Seductive Science - Ember Moans" || b.Title != "Seductive Science - Final Exam" {
		t.Errorf("titles %q / %q", a.Title, b.Title)
	}
	if a.SceneID == b.SceneID {
		t.Errorf("two endings of one game share scene ID %q; saving one would overwrite the other", a.SceneID)
	}
	if again := part("Ember Moans"); again.SceneID != a.SceneID {
		t.Error("the same part must keep its scene ID across runs")
	}
}

func TestRegistrableDomain(t *testing.T) {
	cases := map[string]string{
		"https://cdns127.hqcollect.is/uploads/x.jpg": "hqcollect.is",
		"https://www.ptorrents.com/dezyred.htm":      "ptorrents.com",
		"https://shop.example.co.uk/a":               "example.co.uk",
		"https://pornolab.net/forum/viewtopic.php":   "pornolab.net",
		"not a url": "",
	}
	for in, want := range cases {
		if got := registrableDomain(in); got != want {
			t.Errorf("registrableDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLearnedDomainsNeedTwoSightingsAndCanBeAllowed(t *testing.T) {
	m := map[string]*LearnedDomain{
		"once.com":    {Domain: "once.com", Count: 1},
		"twice.com":   {Domain: "twice.com", Count: 2},
		"allowed.com": {Domain: "allowed.com", Count: 9, Allowed: true},
	}
	got := blockedDomains(m)
	if len(got) != 1 || got[0] != "twice.com" {
		t.Errorf("blocked = %v; want only twice.com", got)
	}
	if !DomainBlocked("https://cdn.twice.com/x", got) {
		t.Error("subdomains of a learned domain should be skipped too")
	}
}

func TestScrapePageFollowsOverviewLink(t *testing.T) {
	const detailPage = `<html><head><title>Sweet Pink Bunny</title></head><body>
		<p>Cleo Vega in Sweet Pink Bunny, 40 minutes.</p></body></html>`
	overviewPage := `<html><body>
		<a href="/scene/sweet-pink-bunny">Sweet Pink Bunny with Cleo Vega</a>
	</body></html>`

	var detailFetches int
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/scene/sweet-pink-bunny" {
			detailFetches++
			w.Write([]byte(detailPage))
			return
		}
		w.Write([]byte(overviewPage))
	}))
	defer site.Close()

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var answer string
		if len(req.Messages) > 0 && strings.Contains(req.Messages[len(req.Messages)-1].Content, "url: "+site.URL+"/scene/sweet-pink-bunny\n") {
			answer = `{"is_scene_page":true,"is_parent_page":false,"part_title":"","page_kind":"official_studio",
				"match_confidence":0.9,"reason":"title and performer match","title":"Sweet Pink Bunny",
				"studio":"ExampleVR","site":"ExampleVR","site_scene_id":"sweet-pink-bunny",
				"cast":["Cleo Vega"],"tags":[],"synopsis":"","released":"2025-03-03","duration_minutes":40,
				"cover_image":-1,"gallery_images":[],"trailer":-1,"follow_link":-1}`
		} else {
			answer = `{"is_scene_page":false,"is_parent_page":false,"part_title":"","page_kind":"store_or_aggregator",
				"match_confidence":0.4,"reason":"an overview listing scenes","title":"","studio":"","site":"",
				"site_scene_id":"","cast":[],"tags":[],"synopsis":"","released":"","duration_minutes":0,
				"cover_image":-1,"gallery_images":[],"trailer":-1,"follow_link":0}`
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message":       map[string]any{"content": answer},
				"finish_reason": "stop",
			}},
		})
	}))
	defer llmServer.Close()

	llm, err := llmscrape.NewClient(llmscrape.LLMConfig{BaseURL: llmServer.URL, Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{
		llm:      llm,
		fetcher:  llmscrape.NewFetcher("test", true, 5*time.Second),
		settings: Settings{MaxPageChars: 5000, MaxFollowRounds: 2},
	}
	res := s.scrapePage(context.Background(), site.URL+"/overview", 7,
		&llmscrape.FileInfo{Filename: "sweet-pink-bunny-cleo-vega.mp4"}, "sweet pink bunny", false)

	if res.Error != "" {
		t.Fatalf("error: %s", res.Error)
	}
	if res.Draft == nil {
		t.Fatalf("no draft was proposed (skipped: %s)", res.Skipped)
	}
	if detailFetches != 1 {
		t.Fatalf("detail page fetched %d times, want 1", detailFetches)
	}
	wantURL := site.URL + "/scene/sweet-pink-bunny"
	if res.URL != wantURL || res.Draft.SourceURL != wantURL {
		t.Errorf("result URL = %q, draft source = %q; want the followed page %q", res.URL, res.Draft.SourceURL, wantURL)
	}
	scene, err := res.Draft.Scene()
	if err != nil {
		t.Fatal(err)
	}
	if scene.HomepageURL != wantURL {
		t.Errorf("homepage = %q, want %q", scene.HomepageURL, wantURL)
	}
	if scene.Title != "Sweet Pink Bunny" {
		t.Errorf("title = %q", scene.Title)
	}
}
