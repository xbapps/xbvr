package scrape

import (
	"strings"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// Someone adds a studio as a custom JAV scraper before XBVR ships it, then XBVR ships it as a preset.
// From then on the presets cover everything the custom one crawls, so it retires:
//   - startup: it isn't loaded
//   - after the migrations: its switches (enabled, subscribed, limit scraping) move to the preset,
//     its Site row goes, and it leaves scrapers.json
//
// Its scenes stay put. With their scraper gone they're orphans, and the preset's next run adopts them
// (only scraper_id changes, nothing is fetched again).

// r18devRetired: custom site ID -> the preset site IDs that cover it. Filled at startup.
var r18devRetired = map[string][]string{}

// r18devCoveredBy returns the presets that between them crawl everything custom crawls, or nil if
// any of it (a source or a content ID) is something no preset does. Quality modes don't count, they
// only take scenes while they're on.
func r18devCoveredBy(custom config.R18devScraperConfig, presets []config.R18devScraperConfig) []string {
	if len(custom.Sources) == 0 && len(custom.ContentIDs) == 0 {
		return nil
	}
	var by []string
	use := func(p config.R18devScraperConfig) {
		id := r18devSiteID(p)
		for _, b := range by {
			if b == id {
				return
			}
		}
		by = append(by, id)
	}
	for _, src := range custom.Sources {
		found := false
		for _, p := range presets {
			for _, ps := range p.Sources {
				if !p.Exclusive && ps == src {
					found = true
					use(p)
				}
			}
		}
		if !found {
			return nil
		}
	}
	for _, cid := range custom.ContentIDs {
		found := false
		for _, p := range presets {
			for _, pc := range p.ContentIDs {
				if !p.Exclusive && strings.EqualFold(pc, cid) {
					found = true
					use(p)
				}
			}
		}
		if !found {
			return nil
		}
	}
	return by
}

// R18devRetireCovered finishes retiring the custom scrapers startup skipped. Needs the database, so it
// runs once the migrations are done.
func R18devRetireCovered() {
	if len(r18devRetired) == 0 {
		return
	}
	db, _ := models.GetCommonDB()

	var picks models.KV
	hasPicks := db.Where(&models.KV{Key: r18devStudioPicksKey}).First(&picks).Error == nil

	for customID, presetIDs := range r18devRetired {
		var old models.Site
		if db.Where(&models.Site{ID: customID}).First(&old).Error == nil {
			for _, pid := range presetIDs {
				var p models.Site
				if db.Where(&models.Site{ID: pid}).First(&p).Error != nil {
					continue
				}
				// a quality mode being on means the custom one is off, so this never breaks a mode
				db.Model(&p).UpdateColumns(map[string]interface{}{
					"is_enabled":     p.IsEnabled || old.IsEnabled,
					"subscribed":     p.Subscribed || old.Subscribed,
					"limit_scraping": p.LimitScraping || old.LimitScraping,
				})
			}
			db.Delete(&models.Site{ID: customID})
		}
		// a quality mode is on and remembers the custom one as a studio pick: remember the presets instead
		if hasPicks {
			var ids []string
			for _, id := range strings.Split(picks.Value, ",") {
				if id == customID {
					ids = append(ids, presetIDs...)
				} else if id != "" {
					ids = append(ids, id)
				}
			}
			picks.Value = strings.Join(ids, ",")
		}
	}
	if hasPicks {
		picks.Save()
	}

	// scrapers.json last: if anything above fails, the next start tries again
	var list config.ScraperList
	list.Load()
	kept := []config.R18devScraperConfig{}
	for _, cfg := range list.CustomScrapers.R18devScrapers {
		presetIDs, retired := r18devRetired[r18devSiteID(cfg)]
		if !retired {
			kept = append(kept, cfg)
			continue
		}
		log.Infof("Custom JAV scraper %s (%s) removed: XBVR now ships %s, which covers everything it did. Its scenes move over on the next run.",
			cfg.Name, r18devSiteID(cfg), strings.Join(presetIDs, ", "))
	}
	list.CustomScrapers.R18devScrapers = kept
	if err := list.Save(); err != nil {
		log.Warnf("Couldn't remove retired custom JAV scrapers from scrapers.json: %v", err)
		return
	}
	r18devRetired = map[string][]string{}
}
