package scrape

import (
	"sync"

	"github.com/mozillazg/go-slugify"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

var reloadMutex sync.Mutex

func reregisterScraper(id string, name string, avatarURL string, domain string, f models.ScraperFunc) {
	models.ReregisterScraper(id, name, avatarURL, domain, f, "")
}

func reregisterAlternateScraper(id string, name string, avatarURL string, domain string, masterSiteId string, f models.ScraperFunc) {
	models.ReregisterScraper(id, name, avatarURL, domain, f, masterSiteId)
}

// applyCustomScrapers registers every custom entry in list, replacing the
// entry (including its scrape closure, so URL/company/master changes apply)
// when the ID is already registered. Returns added/updated counts. It does
// not touch Site rows; ReloadCustomScrapers syncs those via InitSites.
func applyCustomScrapers(list *config.ScraperList) (added int, updated int) {
	apply := func(wantID string, add func()) {
		if _, exists := models.GetScraperByID(wantID); exists {
			updated++
		} else {
			added++
		}
		add()
	}

	for _, scraper := range list.CustomScrapers.SlrScrapers {
		s := scraper
		apply(s.ID, func() {
			addSLRScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
		})
	}
	for _, scraper := range list.CustomScrapers.VrpornScrapers {
		s := scraper
		apply(s.ID, func() {
			addVRPornScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
		})
	}
	for _, scraper := range list.CustomScrapers.PovrScrapers {
		s := scraper
		apply(s.ID, func() {
			addPOVRScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
		})
	}
	for _, scraper := range list.CustomScrapers.VrphubScrapers {
		s := scraper
		apply(s.ID, func() {
			addVRPHubScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, noop)
		})
	}
	for _, scraper := range list.CustomScrapers.StashDbScrapers {
		s := scraper
		// addStashScraper derives the registered ID from the name, not scraper.ID
		apply(slugify.Slugify(s.Name)+"-stashdb", func() {
			addStashScraper(slugify.Slugify(s.Name), s.Name, s.AvatarUrl, s.URL, s.MasterSiteId)
		})
	}
	for _, scraper := range list.CustomScrapers.RealVRScrapers {
		s := scraper
		apply(s.ID, func() {
			addRealVRScraper(s.ID, s.Name, s.Company, s.AvatarUrl, true, s.URL, s.MasterSiteId)
		})
	}
	return added, updated
}

// ReloadCustomScrapers re-reads scrapers.json and activates new or edited
// custom sites without a restart: new IDs are registered, existing IDs are
// re-registered in place (fresh closure, so rebinds and renames apply to
// future scrapes), then Site rows are synced. Entries deleted from the file
// stay registered until restart. Serialized; avoid calling mid-scrape.
func ReloadCustomScrapers() (added int, updated int) {
	reloadMutex.Lock()
	defer reloadMutex.Unlock()

	var list config.ScraperList
	list.Load()
	added, updated = applyCustomScrapers(&list)
	models.InitSites()
	return added, updated
}
