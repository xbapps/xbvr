package scrape

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func LittleCaprice(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "littlecaprice"
	siteID := "Little Caprice Dreams"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.littlecaprice-dreams.com")
	siteCollector := createCollector("www.littlecaprice-dreams.com")
	galleryCollector := cloneCollector(sceneCollector)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Little Caprice Media"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - Generate randomly
		e.ForEach(`link[rel="shortlink"]`, func(id int, e *colly.HTMLElement) {
			link := e.Request.AbsoluteURL(e.Attr("href"))
			tmpurl, _ := url.Parse(link)
			sc.SiteID = tmpurl.Query().Get("p")
		})
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Title
		e.ForEach(`.project-header h1`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = strings.TrimSpace(e.Text)
			}
		})

		// Cover
		e.ForEach(`meta[name="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Duration

		// Released
		e.ForEach(`meta[name="og:published_time"]`, func(id int, e *colly.HTMLElement) {
			dt, _ := time.Parse("2006-01-02", e.Attr("content")[:10])
			sc.Released = dt.Format("2006-01-02")
		})

		// Synopsis
		e.ForEach(`.desc-text`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Cast and tags
		e.ForEach(`.project-models .list a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Tags
		e.ForEach(`meta[name="og:video:tag"]`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Attr("content"))
		})

		// Gallery
		out <- sc
	})

	galleryCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)

		e.ForEach(`.et_pb_gallery_items.et_post_gallery .et_pb_gallery_item a`, func(id int, e *colly.HTMLElement) {
			image := strings.Replace(e.Attr("href"), "media.", "www.", -1)
			sc.Gallery = append(sc.Gallery, image)
		})

		out <- sc
	})

	siteCollector.OnHTML(`.project-preview`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exists in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			//sceneCollector.Visit(sceneURL)
			sceneCollector.Request("GET", sceneURL, nil, nil, nil)
		}
	})

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("cover", "")
		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)
	} else {
		siteCollector.Visit("https://www.littlecaprice-dreams.com/collection/virtual-reality/")
	}

	// Missing "Me and You" (my-first-time) scene
	sceneURL := "https://www.littlecaprice-dreams.com/project/vr-180-little-caprice-my-first-time/"
	if !funk.ContainsString(knownScenes, sceneURL) {
		ctx := colly.NewContext()
		ctx.Put("cover", "https://www.littlecaprice-dreams.com/wp-content/uploads/2021/08/wpp_Little-Caprice-Virtual-Reality_.jpg")

		//sceneCollector.Visit(sceneURL)
		sceneCollector.Visit(sceneURL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("littlecaprice", "Little Caprice Dreams", "https://www.littlecaprice-dreams.com/wp-content/uploads/2019/03/cropped-lcd-heart-192x192.png", "littlecaprice-dreams.com", LittleCaprice)
}
