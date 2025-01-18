package scrape

import (
	"strings"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func addRealVRScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string, masterSiteId string) {
	suffixedName := name
	siteNameSuffix := name
	if custom {
		suffixedName += " (Custom RealVR)"
		siteNameSuffix += " (RealVR)"
	} else {
		suffixedName += " (RealVR)"
	}
	if avatarURL == "" {
		avatarURL = "https://realvr.com/icons/realvr/favicon-32x32.png"
	}

	siteURL = strings.TrimSuffix(siteURL, "/")
	siteURL += "/videos/1?order=newest"

	if masterSiteId == "" {
		registerScraper(id, suffixedName, avatarURL, "realvr.com", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, "", false)
		})
	} else {
		registerAlternateScraper(id, suffixedName, avatarURL, "realvr.com", masterSiteId, func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, masterSiteId, false)
		})
	}
}

func init() {
	registerScraper("realvr-single_scene", "RealVR - Other Studios", "", "realvr.com", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "", "", "", "", singeScrapeAdditionalInfo, limitScraping, "", false)
	})
	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.RealVRScrapers {
		addRealVRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL, scraper.MasterSiteId)
	}
	for _, scraper := range scrapers.CustomScrapers.RealVRScrapers {
		addRealVRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, true, scraper.URL, scraper.MasterSiteId)
	}
}
