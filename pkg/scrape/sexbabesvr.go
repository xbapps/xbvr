package scrape

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func SexBabesVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "sexbabesvr"
	siteID := "SexBabesVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("sexbabesvr.com")
	siteCollector := createCollector("sexbabesvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "SexBabesVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID -
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.SiteID = e.Attr("data-scene")
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
			sc.Covers = append(sc.Covers, e.Attr("poster"))
		})

		// Title
		e.ForEach(`div.video-detail__description--container h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Gallery
		e.ForEach(`.gallery-slider img`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("src")))
		})

		// Synopsis
		e.ForEach(`div.video-detail>div.container>p`, func(id int, e *colly.HTMLElement) {
			// Handle blank <p></p> surrounding the synopsis
			if strings.TrimSpace(e.Text) != "" {
				sc.Synopsis = strings.TrimSpace(e.Text)
			}
		})

		// Tags
		e.ForEach(`.tags a.tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		e.ForEach(`div.video-detail__description--author a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`div.video-detail__description--container div`, func(id int, e *colly.HTMLElement) {
			// no good selector, loop through divs for a pattern match
			var re = regexp.MustCompile(`(?m)^\d{2}\/\d{2}\/\d{4}$`)
			match := re.FindAllString(e.Text, -1)
			if len(match) > 0 {
				tmpDate, err := time.Parse("01/02/2006", e.Text)
				if err == nil {
					sc.Released = tmpDate.Format("2006-01-02")
				}
			}
		})

		// Duration

		// Filenames
		// old site,  needs update
		e.ForEach(`div.modal a.vd-row`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("href"))
			base := origURL.Query().Get("response-content-disposition")
			base = strings.Replace(base, "attachment; filename=", "", -1)
			base = strings.Replace(base, "\"", "", -1)
			base = strings.Replace(base, "_trailer", "", -1)
			if !funk.ContainsString(sc.Filenames, base) {
				sc.Filenames = append(sc.Filenames, base)
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`a.pagination__button`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`a.video-container__description--title`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://sexbabesvr.com/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("sexbabesvr", "SexBabesVR", "https://sexbabesvr.com/assets/front/assets/logo.png", SexBabesVR)
}
