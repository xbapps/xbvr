package scrape

import (
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func StasyQVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "stasyqvr"
	siteID := "StasyQVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("stasyqvr.com")
	siteCollector := createCollector("stasyqvr.com")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "StasyQ"
		sc.Site = siteID
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

		// trailer details
		sc.TrailerType = "deovr"
		sc.TrailerSrc = `http://stasyqvr.com/deovr_feed/json/id/` + sc.SiteID

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.video-info div.model-one a`, func(id int, e *colly.HTMLElement) {
			name := ""
			imgUrl := ""
			e.ForEach(`h2`, func(id int, e *colly.HTMLElement) {
				name = strings.TrimSpace(e.Text)
				sc.Cast = append(sc.Cast, name)
			})
			e.ForEach(`img`, func(id int, e *colly.HTMLElement) {
				imgUrl = e.Attr("src")
			})
			sc.ActorDetails[name] = models.ActorDetails{ProfileUrl: e.Request.AbsoluteURL(e.Attr("href")), ImageUrl: imgUrl}
		})

		// Date
		e.ForEach(`div.video-meta-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		sc.Duration = e.Request.Ctx.GetAny("duration").(int)

		// Filenames
		e.ForEach(`div.video-download a.vd-row`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("href"))
			base := origURL.Query().Get("response-content-disposition")
			base = strings.Replace(base, "attachment; filename=", "", -1)
			base = strings.Replace(base, "\"", "", -1)
			if !funk.ContainsString(sc.Filenames, base) {
				sc.Filenames = append(sc.Filenames, base)
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination div.select-links a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`section.grid div.grid-info-inner`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))
		duration, err := strconv.Atoi(strings.Split(e.ChildText(`span:first-of-type`), " ")[0])
		if err != nil {
			duration = 0
		}

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			ctx := colly.NewContext()
			ctx.Put("duration", duration)
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("duration", 0)
		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)
	} else {
		siteCollector.Visit("https://stasyqvr.com/virtualreality/list")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("stasyqvr", "StasyQVR", "https://stasyqvr.com/s/images/apple-touch-icon.png", "stasyqvr.com", StasyQVR)
}
