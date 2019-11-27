package scrape

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func WetVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "wetvr"
	siteID := "WetVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.wetvr.com")
	siteCollector := createCollector("www.wetvr.com")

	sceneCollector.OnHTML(`div#t2019`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "WetVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`h1.t2019-stitle`))

		// Date & Duration
		e.ForEach(`div#t2019-stime span`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				tmpDate, _ := goment.New(e.Text, "MMMM DD, YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
			if id == 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Text, "minutes", "", -1)))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// Cover URLs
		coverSrc := e.ChildAttr(`div#t2019-video deo-video`, "cover-image")
		if coverSrc == "" {
			coverSrc = e.ChildAttr(`div#t2019-video img#no-player-image`, "src")
		}
		if coverSrc != "" {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(coverSrc))
		}

		// Gallery
		e.ForEach(`div.t2019-thumbs img`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("src")))
			}
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div#t2019-description`))

		// Cast
		e.ForEach(`div#t2019-models a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Tags
		// no tags on this site

		// Filenames
		// NOTE: no way to guess filename

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination a.page-link`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.card > a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.wetvr.com/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("wetvr", "WetVR", "https://wetvr.com/images/sites/wetvr/icon-md-f17eedf082.png", WetVR)
}
