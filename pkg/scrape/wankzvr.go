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

func WankzVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "wankzvr"
	siteID := "WankzVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.wankzvr.com")
	siteCollector := createCollector("www.wankzvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Wankz"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`header h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = e.Text
		})

		// Date
		e.ForEach(`div.date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "DD MMMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.duration`, func(id int, e *colly.HTMLElement) {
			if id == 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Text, "minutes", "", -1)))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// Filenames
		base := e.Request.URL.Path
		base = strings.Replace(base, "/", "", -1)
		base = strings.Replace(base, sc.SiteID, "", -1)
		sc.Filenames = append(sc.Filenames, "wankzvr-"+base+"180_180x180_3dh_LR.mp4")

		// Cover URLs
		e.ForEach(`div.swiper-slide img`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("src")))
			}
		})

		// Gallery
		e.ForEach(`div.swiper-slide img.lazyload`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("data-src")))
			}
		})

		// Synopsis
		e.ForEach(`p.description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(strings.Replace(e.Text, " Read more", "", -1))
		})

		// Tags
		e.ForEach(`div.tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast
		e.ForEach(`header h4 a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		out <- sc
	})

	siteCollector.OnHTML(`nav.pager a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.contentContainer article a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		sceneURL = strings.Replace(sceneURL, "/preview", "", -1)

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.wankzvr.com/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("wankzvr", "WankzVR", "https://twivatar.glitch.me/wankzvr", WankzVR)
}
