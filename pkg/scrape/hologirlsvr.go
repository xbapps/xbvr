package scrape

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func HoloGirlsVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "hologirlsvr"
	siteID := "HoloGirlsVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.hologirlsvr.com")
	siteCollector := createCollector("www.hologirlsvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "HoloFilm Productions"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.video-title h3`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = e.Text
			}
		})

		// Cover URLs
		e.ForEach(`div.vidCover`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.TrimSpace(e.ChildAttr(`img`, "src")))
			}
		})

		// Gallery
		e.ForEach(`div.vid-flex-container a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(e.Attr("href")))
		})

		// Synopsis
		r, _ := regexp.Compile("(?s)Synopsis:(.*)Tags:")
		e.ForEach(`div.vidpage-info`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				match := r.FindStringSubmatch(e.Text)
				sc.Synopsis = strings.TrimSpace(match[1])
			}
		})

		// Cast
		e.ForEach(`div.vidpage-featuring span`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Tags
		e.ForEach(`div.videopage-tags em`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination-container li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.memVid div.coverPrev a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.hologirlsvr.com/Scenes")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("hologirlsvr", "HoloGirlsVR", "https://twivatar.glitch.me/hologirlsvr", HoloGirlsVR)
}
