package models

import (
	"testing"
)

func TestReregisterScraper(t *testing.T) {
	// replace-in-place keeps registry order and size, swaps the binding
	RegisterScraper("rereg-test-zzz", "Zzz", "", "example.com", nil, "")
	n := len(scrapers)
	ReregisterScraper("rereg-test-zzz", "Zzz Renamed", "", "example.com", nil, "some-master")
	if len(scrapers) != n {
		t.Fatalf("reregister appended a duplicate: size %d, want %d", len(scrapers), n)
	}
	s, ok := GetScraperByID("rereg-test-zzz")
	if !ok || s.Name != "Zzz Renamed" || s.MasterSiteId != "some-master" {
		t.Errorf("reregister did not replace in place: %+v", s)
	}

	// absent ID appends
	ReregisterScraper("rereg-test-yyy", "Yyy", "", "example.com", nil, "")
	if len(scrapers) != n+1 {
		t.Fatalf("reregister of new ID did not append: size %d, want %d", len(scrapers), n+1)
	}
	if _, ok := GetScraperByID("rereg-test-yyy"); !ok {
		t.Error("appended scraper not found by ID")
	}
	if _, ok := GetScraperByID("rereg-test-nope"); ok {
		t.Error("GetScraperByID reported an unknown ID")
	}
}
