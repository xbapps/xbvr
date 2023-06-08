package scrape

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

var currentYear int
var lastMonth int

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
		sc.Released = e.Request.Ctx.Get("released")

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

	type video struct {
		url      string
		released string
	}
	videoList := make(map[int]video)

	siteCollector.OnHTML(`div.videos__content`, func(e *colly.HTMLElement) {
		e.ForEach(`a.video-container__description--information`, func(cnt int, e *colly.HTMLElement) {
			sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
			var re = regexp.MustCompile(`(?m)(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{2}`)
			match := re.FindAllString(e.Text, -1)

			if len(match) > 0 {
				// If scene exist in database, there's no need to scrape
				page, _ := strconv.Atoi(strings.ReplaceAll(e.Request.URL.String(), "https://sexbabesvr.com/videos/", ""))
				videoList[page*1000+cnt] = video{url: sceneURL, released: match[0]}
			}
		})
	})

	currentYear = time.Now().Year()
	lastMonth = int(time.Now().Month())

	siteCollector.Visit("https://sexbabesvr.com/videos")

	// Sort the videoList as page visits may not return in the same speed and be out of order
	var sortedVideos []int
	for key := range videoList {
		sortedVideos = append(sortedVideos, key)
	}
	sort.Ints(sortedVideos)

	for _, seq := range sortedVideos {
		ctx := colly.NewContext()
		tmpDate, _ := time.Parse("Jan 02", videoList[seq].released)
		if tmpDate.Month() == 12 && lastMonth == 1 {
			currentYear -= 1
		} else if tmpDate.Month() == 1 && lastMonth == 12 {
			currentYear += 1
		}
		tmpDate = tmpDate.AddDate(currentYear-tmpDate.Year(), 0, 0)
		lastMonth = int(tmpDate.Month())
		ctx.Put("released", tmpDate.Format("2006-01-02"))

		if !funk.ContainsString(knownScenes, videoList[seq].url) {
			sceneCollector.Request("GET", videoList[seq].url, nil, ctx, nil)
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("sexbabesvr", "SexBabesVR", "https://sexbabesvr.com/assets/front/assets/logo.png", SexBabesVR)
}
