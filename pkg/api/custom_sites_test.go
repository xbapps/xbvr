package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
)

func TestCustomAggregatorForURL(t *testing.T) {
	cases := []struct {
		url   string
		want  string
		wantOK bool
	}{
		{"https://www.sexlikereal.com/studios/foo", "slr", true},
		{"https://vrporn.com/studio/foo", "vrporn", true},
		{"https://povr.com/studios/foo", "povr", true},
		{"https://vrphub.com/category/foo", "vrphub", true},
		{"https://stashdb.org/studios/1", "stashdb", true},
		{"https://realvr.com/vrpornstudio/foo-1", "realvr", true},
		{"https://virtualrealporn.com/scenes/1", "", false},
		{"https://example.com/studios/foo", "", false},
		{"not a url", "", false},
	}
	for _, c := range cases {
		got, ok := customAggregatorForURL(c.url)
		if got != c.want || ok != c.wantOK {
			t.Errorf("customAggregatorForURL(%q) = (%q, %v), want (%q, %v)", c.url, got, ok, c.want, c.wantOK)
		}
	}
}

func TestFlattenCustomScraperGroups(t *testing.T) {
	groups := map[string][]config.ScraperConfig{
		"slr": {
			{URL: "https://www.sexlikereal.com/studios/foo", Name: "Foo", Company: "Foo", MasterSiteId: "foo-main"},
		},
		"vrporn": {
			{URL: "https://vrporn.com/studio/bar", Name: "Bar", Company: "Bar Co"},
		},
		"povr":    {},
		"vrphub":  {},
		"stashdb": {},
		"realvr":  {},
	}

	entries := flattenCustomScraperGroups(groups)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Aggregator != "slr" || entries[0].MasterSiteId != "foo-main" {
		t.Errorf("first entry should be the slr site with its master binding, got %+v", entries[0])
	}
	if entries[1].Aggregator != "vrporn" || entries[1].URL != "https://vrporn.com/studio/bar" {
		t.Errorf("second entry should be the vrporn site, got %+v", entries[1])
	}
}

func TestCustomScraperGroupsRoundTrip(t *testing.T) {
	oldAppDir := common.AppDir
	common.AppDir = t.TempDir()
	defer func() { common.AppDir = oldAppDir }()

	seed := `{"warning": [], "custom": {"slr": [{"url": "https://www.sexlikereal.com/studios/foo", "name": "Foo", "company": "Foo"}], "povr": [], "stashdb": [], "realvr": [], "vrporn": [], "vrphub": []}, "xbvr": {}}`
	if err := os.WriteFile(filepath.Join(common.AppDir, "scrapers.json"), []byte(seed), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, groups := customScraperGroups()
	if len(groups["slr"]) != 1 || groups["slr"][0].MasterSiteId != "" {
		t.Fatalf("expected one unbound slr entry, got %+v", groups["slr"])
	}

	// bind a main site, save, reload — this is the "forgot the parent" fix-up
	groups["slr"][0].MasterSiteId = "foo-main"
	if err := saveCustomScraperGroups(cfg, groups); err != nil {
		t.Fatal(err)
	}

	_, reloaded := customScraperGroups()
	entries := flattenCustomScraperGroups(reloaded)
	if len(entries) != 1 || entries[0].MasterSiteId != "foo-main" {
		t.Fatalf("master binding did not survive save/reload, got %+v", entries)
	}

	// delete the entry, save, reload
	reloaded["slr"] = reloaded["slr"][:0]
	if err := saveCustomScraperGroups(cfg, reloaded); err != nil {
		t.Fatal(err)
	}
	_, afterDelete := customScraperGroups()
	if len(flattenCustomScraperGroups(afterDelete)) != 0 {
		t.Fatalf("entry was not deleted, got %+v", flattenCustomScraperGroups(afterDelete))
	}
}
