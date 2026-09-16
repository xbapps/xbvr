package scrape

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func writeSeedScrapers(t *testing.T, seed string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(common.AppDir, "scrapers.json"), []byte(seed), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyCustomScrapers(t *testing.T) {
	oldAppDir := common.AppDir
	common.AppDir = t.TempDir()
	defer func() { common.AppDir = oldAppDir }()

	seed := `{"warning": [], "custom": {
		"slr": [{"url": "https://www.sexlikereal.com/studios/reloadtestzzz", "name": "ReloadTestZzz", "company": "ReloadTestZzz"}],
		"vrporn": [{"url": "https://vrporn.com/studio/reloadtestyyy", "name": "ReloadTestYyy", "company": "ReloadTestYyy", "master_site_id": "vrbangers"}],
		"povr": [], "stashdb": [], "realvr": [], "vrphub": []}, "xbvr": {}}`
	writeSeedScrapers(t, seed)

	var list config.ScraperList
	if err := list.Load(); err != nil {
		t.Fatal(err)
	}

	added, updated := applyCustomScrapers(&list)
	if added != 2 || updated != 0 {
		t.Fatalf("first apply = (%d added, %d updated), want (2, 0)", added, updated)
	}

	s, ok := models.GetScraperByID("reloadtestzzz-slr")
	if !ok || s.Scrape == nil {
		t.Fatalf("slr custom not registered: %+v", s)
	}
	if s.MasterSiteId != "" {
		t.Errorf("expected no master binding, got %q", s.MasterSiteId)
	}
	v, ok := models.GetScraperByID("reloadtestyyy-vrporn")
	if !ok || v.MasterSiteId != "vrbangers" {
		t.Errorf("vrporn custom master not applied: %+v", v)
	}

	// bind the forgotten parent, add one more site, re-apply: no restart
	seed2 := `{"warning": [], "custom": {
		"slr": [{"url": "https://www.sexlikereal.com/studios/reloadtestzzz", "name": "ReloadTestZzz", "company": "ReloadTestZzz", "master_site_id": "vrbangers"}],
		"vrporn": [{"url": "https://vrporn.com/studio/reloadtestyyy", "name": "ReloadTestYyy", "company": "ReloadTestYyy", "master_site_id": "vrbangers"}],
		"povr": [{"url": "https://povr.com/studios/reloadtestxxx", "name": "ReloadTestXxx", "company": "ReloadTestXxx"}],
		"stashdb": [], "realvr": [], "vrphub": []}, "xbvr": {}}`
	writeSeedScrapers(t, seed2)

	var list2 config.ScraperList
	if err := list2.Load(); err != nil {
		t.Fatal(err)
	}
	added, updated = applyCustomScrapers(&list2)
	if added != 1 || updated != 2 {
		t.Fatalf("second apply = (%d added, %d updated), want (1, 2)", added, updated)
	}

	s, ok = models.GetScraperByID("reloadtestzzz-slr")
	if !ok || s.MasterSiteId != "vrbangers" || s.Scrape == nil {
		t.Errorf("rebind did not take effect without restart: %+v", s)
	}
	if _, ok := models.GetScraperByID("reloadtestxxx-povr"); !ok {
		t.Error("new povr custom not registered on reload")
	}
}
