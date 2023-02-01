package scrape

import (
	"regexp"
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

	sceneCollector := createCollector("wetvr.com")
	siteCollector := createCollector("wetvr.com")

	// RegEx Patterns
	durationRegEx := regexp.MustCompile(`(?i)DURATION:\W(\d+)`)

	sceneCollector.OnHTML(`div#t2019`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "WetVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://wetvr.com/", "https://members.wetvr.com/", 1)

		// Scene ID - get from previous page
		sc.SiteID = e.Request.Ctx.GetAny("scene-id").(string)
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`h1.t2019-stitle`))

		// Date
		scenedate := e.Request.Ctx.GetAny("scene-date").(string)
		tmpDate, _ := goment.New(scenedate, "MMMM DD, YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		// Duration
		tmpDuration := durationRegEx.FindStringSubmatch(e.ChildText(`div#t2019-stime`))[1]
		sc.Duration, _ = strconv.Atoi(tmpDuration)

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

		// trailer details
		sc.TrailerType = "deovr"
		sc.TrailerSrc = strings.Replace(sc.HomepageURL, "/video/", "/deovr/", 1)

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

	siteCollector.OnHTML(`div.card`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a", "href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {

			// SceneID and release date are only available here on div.card
			ctx := colly.NewContext()
			ctx.Put("scene-id", e.Attr("data-video-id"))
			ctx.Put("scene-date", e.Attr("data-date"))

			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	siteCollector.Visit("https://wetvr.com/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("wetvr", "WetVR", "https://wetvr.com/assets/images/sites/wetvr/logo-4a2f06a4c9.png", WetVR)
}
