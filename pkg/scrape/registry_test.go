package scrape

import (
	"strings"
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// These tests are hermetic - no network, no fixtures. They guard the registration
// wiring in each scraper's init(), which is easy to break because it is spread
// across ~45 files and only surfaces at runtime.

func TestScraperIDsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, s := range models.GetScrapers() {
		if prev, dup := seen[s.ID]; dup {
			t.Errorf("duplicate scraper ID %q registered by both %q and %q", s.ID, prev, s.Name)
			continue
		}
		seen[s.ID] = s.Name
	}
}

func TestScraperFieldsArePopulated(t *testing.T) {
	scrapers := models.GetScrapers()
	if len(scrapers) == 0 {
		t.Fatal("no scrapers registered - init() wiring is broken")
	}
	for _, s := range scrapers {
		if s.ID == "" {
			t.Errorf("scraper %q has an empty ID", s.Name)
		}
		if s.Name == "" {
			t.Errorf("scraper %q has an empty Name", s.ID)
		}
		if s.Domain == "" {
			t.Errorf("scraper %q has an empty Domain", s.ID)
		}
		if s.Scrape == nil {
			t.Errorf("scraper %q has a nil Scrape func", s.ID)
		}
	}
}

// A scraper whose Domain does not reduce to a usable core domain silently loses its
// rate limit and its per-site header/cookie config in createCollector.
func TestScraperDomainsReduceToCoreDomain(t *testing.T) {
	for _, s := range models.GetScrapers() {
		if s.Domain == "" {
			continue
		}
		core := GetCoreDomain(s.Domain)
		if core == "" {
			t.Errorf("scraper %q: domain %q reduced to an empty core domain", s.ID, s.Domain)
		}
		if strings.Contains(core, "/") {
			t.Errorf("scraper %q: core domain %q still contains a path separator", s.ID, core)
		}
	}
}

// Site-family scrapers (POVR, SLR, StashDB, RealVR, VRPorn, VRPHub) are registered by
// looping over the embedded scrapers.json. If an entry stops registering - a renamed
// JSON key, a changed ID rule - the site quietly disappears from the UI with no error.
func TestOfficialScraperListIsFullyRegistered(t *testing.T) {
	var list config.ScraperList
	if err := list.Load(); err != nil {
		t.Fatalf("loading scraper list: %v", err)
	}

	registered := map[string]bool{}
	for _, s := range models.GetScrapers() {
		registered[s.ID] = true
	}

	families := map[string][]config.ScraperConfig{
		"povr":    list.XbvrScrapers.PovrScrapers,
		"slr":     list.XbvrScrapers.SlrScrapers,
		"stashdb": list.XbvrScrapers.StashDbScrapers,
		"realvr":  list.XbvrScrapers.RealVRScrapers,
		"vrporn":  list.XbvrScrapers.VrpornScrapers,
		"vrphub":  list.XbvrScrapers.VrphubScrapers,
	}

	checked := 0
	for family, entries := range families {
		// Some families ship with no list-driven entries (their sites are registered
		// directly in the scraper's init instead). That is a valid state, not a failure.
		if len(entries) == 0 {
			t.Logf("family %q has no list-driven entries - skipping", family)
			continue
		}
		checked += len(entries)
		missing := 0
		for _, e := range entries {
			if e.ID == "" {
				t.Errorf("family %q: entry with URL %q has no ID assigned", family, e.URL)
				continue
			}
			if !registered[e.ID] {
				missing++
				if missing <= 5 {
					t.Errorf("family %q: scrapers.json entry %q is not registered", family, e.ID)
				}
			}
		}
		if missing > 5 {
			t.Errorf("family %q: %d entries total are not registered", family, missing)
		}
	}

	if checked == 0 {
		t.Fatal("no scrapers.json entries were checked - the embedded list failed to load")
	}
	t.Logf("verified %d list-driven scraper entries are registered", checked)
}
