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

func StasyQVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	logScrapeStart("stasyqvr", "StasyQVR")

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("stasyqvr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
		colly.MaxDepth(5),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("stasyqvr.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "StasyQ"
		sc.Site = "StasyQVR"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = strings.Split(tmp[len(tmp)-1], "-")[0]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.video-title h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover
		e.ForEach(`div.splash-screen`, func(id int, e *colly.HTMLElement) {
			base := e.Attr("style")
			base = strings.Split(base, "background-image: url(")[1]
			base = strings.Split(base, ");")[0]
			base = strings.Split(base, "?")[0]
			sc.Covers = append(sc.Covers, base)
		})

		// Gallery
		e.ForEach(`div.video-gallery figure a`, func(id int, e *colly.HTMLElement) {
			base := e.Request.AbsoluteURL(e.Attr("href"))
			base = strings.Split(base, "?")[0]
			sc.Gallery = append(sc.Gallery, base)
		})

		// Synopsis
		e.ForEach(`div.video-info p`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		// NOTE: StasyQVR doesn't provide tags

		// Cast
		e.ForEach(`div.video-info div.model-one a h2`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`div.video-meta-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		// NOTE: StasyQVR doesn't provide duration information on scene page

		// Filenames
		e.ForEach(`div.video-download a.vd-row`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("href"))
			base := origURL.Query().Get("response-content-disposition")
			base = strings.Replace(base, "attachment; filename=", "", -1)
			base = strings.Replace(base, "\"", "", -1)
			if !funk.ContainsString(sc.Filenames, base) {
				sc.Filenames = append(sc.Filenames, base)
				sc.Filenames = append(sc.Filenames, strings.Replace(base, "original_", "original_"+sc.SiteID, -1))
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination div.select-links a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`section.grid div.grid-info-inner a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if strings.Contains(sceneURL, "scene") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				sceneCollector.Visit(sceneURL)
			}
		}
	})

	siteCollector.Visit("https://stasyqvr.com/virtualreality/list")

	if updateSite {
		updateSiteLastUpdate("stasyqvr")
	}
	logScrapeFinished("stasyqvr", "StasyQVR")
	return nil
}

func init() {
	registerScraper("stasyqvr", "StasyQVR", StasyQVR)
}
