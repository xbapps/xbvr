//go:build scrapelive

// Live scraper smoke tests. Excluded from every normal build and from `go test ./...`
// unless -tags=scrapelive is passed, because they make real requests to the sites.
//
// Run:
//
//	export XBVR_SCRAPE_TARGETS=~/xbvr-scrape-targets.json
//	go test -tags='json1 scrapelive' -vet=off ./pkg/scrape/ -run TestLiveScrape -v
//
// The targets file lives OUTSIDE this repository - it holds scene URLs, which must
// never be committed. Format:
//
//	[
//	  {"scraper": "virtualrealporn", "url": "https://.../vr-porn-video/<slug>/"},
//	  {"scraper": "povr-single_scene", "url": "https://.../vr-porn/<slug>/", "skipCast": true}
//	]
//
// What this catches is the failure mode a site redesign produces: every selector stops
// matching, the scraper emits nothing, and no error is raised anywhere. Assertions stay
// deliberately shape-based (non-empty, plausible range) rather than comparing to exact
// titles or performer names, so the test does not encode that content and does not break
// when a site legitimately edits its own metadata.

package scrape

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/xbapps/xbvr/pkg/models"
)

type liveTarget struct {
	Scraper string `json:"scraper"`
	URL     string `json:"url"`
	// Opt-outs for sites that legitimately omit a field.
	SkipCast    bool `json:"skipCast"`
	SkipTags    bool `json:"skipTags"`
	SkipCover   bool `json:"skipCover"`
	SkipRelease bool `json:"skipRelease"`
}

const liveScrapeTimeout = 3 * time.Minute

func loadLiveTargets(t *testing.T) []liveTarget {
	t.Helper()

	path := os.Getenv("XBVR_SCRAPE_TARGETS")
	if path == "" {
		t.Skip("XBVR_SCRAPE_TARGETS not set - see the comment at the top of live_test.go")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var targets []liveTarget
	if err := json.Unmarshal(raw, &targets); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if len(targets) == 0 {
		t.Fatalf("%s contains no targets", path)
	}
	return targets
}

func findScraper(t *testing.T, id string) models.Scraper {
	t.Helper()
	for _, s := range models.GetScrapers() {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no scraper registered with ID %q", id)
	return models.Scraper{}
}

// runScraperOnce drives one scraper against one scene URL and returns everything it
// emitted. A scraper that matches nothing returns an empty slice rather than an error,
// which is exactly the silent failure we are testing for.
func runScraperOnce(t *testing.T, s models.Scraper, url string) []models.ScrapedScene {
	t.Helper()

	out := make(chan models.ScrapedScene, 16)
	errc := make(chan error, 1)

	var wg models.ScrapeWG
	wg.Add(1)
	go func() {
		// updateSite=false so the test never writes site.LastUpdate;
		// limitScraping=true so a single-scene run cannot walk into pagination.
		errc <- s.Scrape(&wg, false, []string{}, out, url, "", true)
	}()

	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("scraper %q returned an error: %v", s.ID, err)
		}
	case <-time.After(liveScrapeTimeout):
		t.Fatalf("scraper %q timed out after %s", s.ID, liveScrapeTimeout)
	}

	close(out)
	var scenes []models.ScrapedScene
	for sc := range out {
		scenes = append(scenes, sc)
	}
	return scenes
}

func TestLiveScrape(t *testing.T) {
	for _, target := range loadLiveTargets(t) {
		t.Run(target.Scraper, func(t *testing.T) {
			s := findScraper(t, target.Scraper)
			scenes := runScraperOnce(t, s, target.URL)

			if len(scenes) == 0 {
				t.Fatalf("scraper %q emitted no scenes for the target URL - "+
					"selectors have most likely stopped matching after a site change", s.ID)
			}
			sc := scenes[0]

			if sc.SceneID == "" {
				t.Error("SceneID is empty - the scene would be dropped before reaching the DB")
			}
			if sc.SiteID == "" {
				t.Error("SiteID is empty")
			}
			if sc.Title == "" {
				t.Error("Title is empty")
			}
			if sc.HomepageURL == "" {
				t.Error("HomepageURL is empty")
			}
			if len(sc.Filenames) == 0 {
				t.Error("Filenames is empty - nothing on disk could ever match this scene")
			}
			if sc.Duration <= 0 || sc.Duration > 600 {
				t.Errorf("Duration %d min is outside the plausible range", sc.Duration)
			}
			if !target.SkipRelease && sc.Released == "" {
				t.Error("Released is empty")
			}
			if !target.SkipCover && len(sc.Covers) == 0 {
				t.Error("Covers is empty")
			}
			if !target.SkipTags && len(sc.Tags) == 0 {
				t.Error("Tags is empty")
			}
			if !target.SkipCast {
				if len(sc.Cast) == 0 {
					t.Error("Cast is empty")
				}
				for _, name := range sc.Cast {
					if _, ok := sc.ActorDetails[name]; !ok {
						t.Errorf("cast member at index %d has no matching ActorDetails entry",
							indexOf(sc.Cast, name))
					}
				}
			}

			// Log shapes, never values - see the no-names rule in CLAUDE.md.
			t.Logf("ok: %d filenames, %d tags, %d cast, %d covers, %d gallery, %d min, trailer=%s",
				len(sc.Filenames), len(sc.Tags), len(sc.Cast), len(sc.Covers),
				len(sc.Gallery), sc.Duration, sc.TrailerType)
		})
	}
}

func indexOf(haystack []string, needle string) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}
