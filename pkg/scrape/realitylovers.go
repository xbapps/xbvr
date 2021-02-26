package scrape

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/resty.v1"
)

func RealityLoversSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("realitylovers.com", "tsvirtuallovers.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "RealityLovers"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID
		sc.SiteID = e.Request.Ctx.Get("id")
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		sc.Covers = append(sc.Covers, strings.Replace(e.Request.Ctx.Get("cover"), "-Small", "-Large", 1))

		// Title
		sc.Title = e.Request.Ctx.Get("title")

		// Release date
		sc.Released = e.Request.Ctx.Get("released")

		// Cast
		e.ForEach(`a[itemprop="actor"]`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Gallery
		e.ForEach(`img.videoClip__Details--galleryItem`, func(id int, e *colly.HTMLElement) {
			imageURL := strings.Replace(strings.Fields(e.Attr("data-big"))[0], "_small", "_large", 1)
			sc.Gallery = append(sc.Gallery, strings.Replace(imageURL, "https:", "http:", 1))
		})

		// Tags
		e.ForEach(`.videoClip__Details__categoryTag`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			sc.Tags = append(sc.Tags, tag)
		})

		// Synopsis
		e.ForEach(`p[itemprop="description"]`, func(id int, e *colly.HTMLElement) {
			reLeadcloseWhtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
			reInsideWhtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
			synopsis := reLeadcloseWhtsp.ReplaceAllString(e.Text, "")
			synopsis = reInsideWhtsp.ReplaceAllString(synopsis, " ")
			synopsis = strings.TrimSuffix(synopsis, " … Read more")
			sc.Synopsis = synopsis
		})

		out <- sc
	})

	// Request scenes via REST API
	r, err := resty.R().
		SetHeader("User-Agent", userAgent).
		SetHeader("content-type", "application/json;charset=UTF-8").
		SetHeader("accept", "application/json, text/plain, */*").
		SetHeader("referer", URL+"videos").
		SetHeader("origin", URL).
		SetHeader("authority", siteID+".com").
		SetBody(`{"searchQuery":"","categoryId":null,"perspective":null,"actorId":null,"offset":"5000","isInitialLoad":true,"sortBy":"NEWEST","videoView":"MEDIUM","device":"DESKTOP"}`).
		Post(URL + "videos/search?hl=1")
	if err == nil || r.StatusCode() == 200 {
		result := gjson.Get(r.String(), "contents")
		result.ForEach(func(key, value gjson.Result) bool {
			sceneURL := URL + value.Get("videoUri").String()
			if !funk.ContainsString(knownScenes, sceneURL) {
				ctx := colly.NewContext()
				cover := strings.Fields(value.Get("mainImageSrcset").String())[0]
				ctx.Put("cover", strings.Replace(cover, "https:", "http:", 1))
				ctx.Put("id", value.Get("id").String())
				ctx.Put("released", value.Get("released").String())
				ctx.Put("title", value.Get("title").String())
				sceneCollector.Request("GET", sceneURL+"?hl=1", nil, ctx, nil)
			}
			return true
		})
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func RealityLovers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return RealityLoversSite(wg, updateSite, knownScenes, out, "realitylovers", "RealityLovers", "https://realitylovers.com/")
}

func TSVirtualLovers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return RealityLoversSite(wg, updateSite, knownScenes, out, "tsvirtuallovers", "TSVirtualLovers", "https://tsvirtuallovers.com/")
}

func init() {
	registerScraper("realitylovers", "RealityLovers", "http://static.rlcontent.com/shared/VR/common/favicons/apple-icon-180x180.png", RealityLovers)
	registerScraper("tsvirtuallovers", "TSVirtualLovers", "http://static.rlcontent.com/shared/TS/common/favicons/apple-icon-180x180.png", TSVirtualLovers)
}
