package tasks

import "github.com/xbapps/xbvr/pkg/scrape"

func ScrapeActors() {
	scrape.GenericActorScrapers()
}
func ScrapeActor(actorId uint, url string) {
	scrape.GenericSingleActorScraper(actorId, url)
}
func ScrapeActorBySite(site string) {
	scrape.GenericActorScrapersBySite(site)
}
