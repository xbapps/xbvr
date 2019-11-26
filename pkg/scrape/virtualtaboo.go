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

func VirtualTaboo(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "virtualtaboo"
	siteID := "VirtualTaboo"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("virtualtaboo.com")
	siteCollector := createCollector("virtualtaboo.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VirtualTaboo"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		e.ForEach(`#player`, func(id int, e *colly.HTMLElement) {
			sc.SiteID = strings.Split(e.Attr("data-poster-index"), ":")[0]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`div.video-detail h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Filenames
		base := strings.Split(e.Request.URL.Path, "/")[2]
		sc.Filenames = append(sc.Filenames, base+"-files-smartphone.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-gear.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-psvr_180_sbs.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-oculus.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-oculus5k.mp4")

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`div.gallery-item:not(.link) a.gallery-image`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(strings.Split(e.Attr("href"), "?")[0]))
		})

		// Synopsis
		e.ForEach(`div.description span.full`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.tag-list a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Cast
		e.ForEach(`div.video-detail .info a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`div.right-info div.info`, func(id int, e *colly.HTMLElement) {
			tmpData := funk.ReverseStrings(strings.Split(e.Text, "\n"))

			tmpDate, _ := goment.New(strings.TrimSpace(tmpData[1]), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")

			sc.Duration, _ = strconv.Atoi(strings.TrimSpace(strings.Replace(tmpData[3], "min", "", -1)))

		})

		// Duration
		e.ForEach(`p.video-duration`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.Split(e.Attr("content"), ":")[1])
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.video-title a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://virtualtaboo.com/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("virtualtaboo", "VirtualTaboo", "https://twivatar.glitch.me/virtualtaboo", VirtualTaboo)
}
