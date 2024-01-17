package scrape

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

func SexBabesVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
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
			sc.Covers = append(sc.Covers, strings.Replace(e.Attr("poster"), "/videoDetail2x", "", -1))
		})

		// Title
		e.ForEach(`div.video-detail__description--container h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Gallery
		e.ForEach(`.gallery-slider a[data-fancybox=gallery]`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
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
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.video-detail__description--author a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Date
		releaseDateText := e.ChildText(`.video-detail__description--container > div:last-of-type`)
		tmpDate, _ := time.Parse("Jan 02, 2006", releaseDateText)
		sc.Released = tmpDate.Format("2006-01-02")

		// Duration

		// Filenames
		// old site, needs update
		e.ForEach(`div.modal a.vd-row`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("href"))
			base := origURL.Query().Get("response-content-disposition")
			base = strings.ReplaceAll(base, "attachment; filename=", "")
			base = strings.ReplaceAll(base, "\"", "")
			base = strings.ReplaceAll(base, "_trailer", "")
			if !funk.ContainsString(sc.Filenames, base) {
				sc.Filenames = append(sc.Filenames, base)
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`a.pagination__button`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.videos__content`, func(e *colly.HTMLElement) {
		e.ForEach(`a.video-container__description--title`, func(cnt int, e *colly.HTMLElement) {
			sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
			if !funk.ContainsString(knownScenes, sceneURL) {
				sceneCollector.Visit(sceneURL)
			}
		})
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://sexbabesvr.com/vr-porn-videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("sexbabesvr", "SexBabesVR", "https://sexbabesvr.com/assets/front/assets/logo.png", "sexbabesvr.com", SexBabesVR)
}
