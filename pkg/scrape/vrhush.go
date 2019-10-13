package scrape

import (
	"net/url"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRHush(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	logScrapeStart("vrhush", "VRHush")

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("vrhush.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("vrhush.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	castCollector := colly.NewCollector(
		colly.AllowedDomains("vrhush.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
		colly.AllowURLRevisit(),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	castCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRHush"
		sc.Site = "VRHush"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		tmp2 := strings.Split(tmp[len(tmp)-1], "_")[0]
		sc.SiteID = strings.Replace(tmp2, "vrh", "", -1)
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`h1.latest-scene-title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("poster")))
		})

		// Gallery
		e.ForEach(`div.owl-carousel img.img-responsive`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("src")))
		})

		// Synopsis
		e.ForEach(`span.full-description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`p.tag-container a.label-tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Cast
		var tmpCast []string
		e.ForEach(`h5.latest-scene-subtitle a`, func(id int, e *colly.HTMLElement) {
			tmpCast = append(tmpCast, e.Attr("href"))
		})

		// Date
		e.ForEach(`div.latest-scene-meta-1 div.text-left`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		sc.Duration = 0

		// Filenames
		e.ForEach(`input.stream-input-box`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("value"))
			sc.Filenames = append(sc.Filenames, origURL.Query().Get("name"))
		})

		ctx := colly.NewContext()
		ctx.Put("scene", &sc)

		for i := range tmpCast {
			castCollector.Request("GET", tmpCast[i], nil, ctx, nil)
		}

		out <- sc
	})

	castCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(*models.ScrapedScene)

		var name string
		e.ForEach(`h1#model-name`, func(id int, e *colly.HTMLElement) {
			name = strings.TrimSpace(e.Text)
		})

		var gender string
		e.ForEach(`ul.model-attributes li`, func(id int, e *colly.HTMLElement) {
			if strings.Split(e.Text, " ")[0] == "Gender" {
				gender = strings.Split(e.Text, " ")[1]
			}
		})

		if gender == "Female" {
			sc.Cast = append(sc.Cast, name)
		}
	})

	siteCollector.OnHTML(`ul.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.row div.col-md-4 p.desc a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://vrhush.com/scenes")

	if updateSite {
		updateSiteLastUpdate("vrhush")
	}
	logScrapeFinished("vrhush", "VRHush")
	return nil
}

func init() {
	registerScraper("vrhush", "VRHush", VRHush)
}
