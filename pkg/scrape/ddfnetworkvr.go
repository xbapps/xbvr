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

func DDFNetworkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "ddfnetworkvr"
	siteID := "DDFNetworkVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("ddfnetworkvr.com")
	siteCollector := createCollector("ddfnetworkvr.com")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "DDFNetwork"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// ID
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.video-title h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			// NOTE: preview image comes in two flavours - preview_vr.jpg and preview.jpg
			sc.Covers = append(sc.Covers, strings.Replace(e.Attr("poster"), "_vr", "", -1))
		})

		// Cover (for older videos)
		e.ForEach(`div.video-box-inner img`, func(id int, e *colly.HTMLElement) {
			if len(sc.Covers) == 0 && id == 0 {
				sc.Covers = append(sc.Covers, strings.Replace(e.Attr("src"), "_vr", "", -1))
			}
		})

		// Gallery
		e.ForEach(`#photoSliderGuest div.card a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis
		e.ForEach(`div.about-text p.box-container`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`ul.tags li`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Cast
		e.ForEach(`div.video-title h2.actors a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`h2.actors time`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`p.duration`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.Split(e.Text, ":")[1])
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames
		// NOTE: no way to guess filename

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination a.page-link`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div#scenesAjaxReplace a.play-on-hover`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://ddfnetworkvr.com/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("ddfnetworkvr", "DDFNetworkVR", "https://pbs.twimg.com/profile_images/1083417183722434560/Ur5xIhqG_200x200.jpg", DDFNetworkVR)
}
