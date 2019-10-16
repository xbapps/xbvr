package scrape

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/resty.v1"
)

func RealityLovers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	logScrapeStart("realitylovers", "RealityLovers")
	const maxRetries = 15

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("realitylovers.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector.OnRequest(func(r *colly.Request) {
		attempt := r.Ctx.GetAny("attempt")

		if attempt == nil {
			r.Ctx.Put("attempt", 1)
		}

		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnError(func(r *colly.Response, err error) {
		attempt := r.Ctx.GetAny("attempt").(int)

		if r.StatusCode == 429 {
			log.Println("Error:", r.StatusCode, err)

			if attempt <= maxRetries {
				unCache(r.Request.URL.String(), sceneCollector.CacheDir)
				log.Println("Waiting 2 seconds before next request...")
				r.Ctx.Put("attempt", attempt+1)
				time.Sleep(2 * time.Second)
				r.Request.Retry()
			}
		}
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "RealityLovers"
		sc.Site = "RealityLovers"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID
		sc.SiteID = e.Request.Ctx.Get("id")
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		sc.Covers = append(sc.Covers, e.Request.Ctx.Get("cover"))

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
			imageURL := strings.Fields(e.Attr("data-big"))[0]
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
		SetHeader("referer", "https://realitylovers.com/videos").
		SetHeader("origin", "https://realitylovers.com").
		SetHeader("authority", "realitylovers.com").
		SetBody(`{"searchQuery":"","categoryId":null,"perspective":null,"actorId":null,"offset":"5000","isInitialLoad":true,"sortBy":"NEWEST","videoView":"MEDIUM","device":"DESKTOP"}`).
		Post("https://realitylovers.com/videos/search")
	if err == nil || r.StatusCode() == 200 {
		result := gjson.Get(r.String(), "contents")
		result.ForEach(func(key, value gjson.Result) bool {
			sceneURL := "https://realitylovers.com/" + value.Get("videoUri").String()
			if !funk.ContainsString(knownScenes, sceneURL) {
				ctx := colly.NewContext()
				cover := strings.Fields(value.Get("mainImageSrcset").String())[0]
				ctx.Put("cover", strings.Replace(cover, "https:", "http:", 1))
				ctx.Put("id", value.Get("id").String())
				ctx.Put("released", value.Get("released").String())
				ctx.Put("title", value.Get("title").String())
				sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
			}
			return true
		})
	}

	if updateSite {
		updateSiteLastUpdate("realitylovers")
	}
	logScrapeFinished("realitylovers", "RealityLovers")
	return nil
}

func init() {
	registerScraper("realitylovers", "RealityLovers", RealityLovers)
}
