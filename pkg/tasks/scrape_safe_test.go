package tasks

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/models"
)

// A panicking scraper must come back as a logged skip, not take down the
// caller — this is what keeps one bad site from killing a whole run (#2279).
// The fake defers wg.Done like every real Scrape does, so the batch counter
// must read zero afterwards with no DB touched (the panic fires first).
func TestRunScraperSafeIsolatesPanic(t *testing.T) {
	var wg models.ScrapeWG
	wg.Add(1)
	s := models.Scraper{
		ID: "panic-test",
		Scrape: func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, collectedScenes chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			defer wg.Done()
			panic("boom")
		},
	}

	runScraperSafe(&wg, s, false, nil, nil, "", "", false)

	if n := wg.Count(); n != 0 {
		t.Fatalf("wg count = %d, want 0", n)
	}
}
