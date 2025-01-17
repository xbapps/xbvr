package scrape

import (
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func KinkVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "kinkvr"
	siteID := "KinkVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("kinkvr.com")
	siteCollector := createCollector("kinkvr.com")

	// These cookies are needed for age verification.
	siteCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "agreedToDisclaimer=true")
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "agreedToDisclaimer=true")
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Badoink"
		sc.Site = siteID
		sc.SiteID = ""
		sc.HomepageURL = e.Request.URL.String()

		// Cover Url
		coverURL := e.ChildAttr("div#povVideoContainer dl8-video", "poster")
		sc.Covers = append(sc.Covers, coverURL)

		// Gallery
		e.ForEach(`div.owl-carousel div.item`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.ChildAttr("img", "src"))
		})

		// Incase we scrape a single scene use one of the gallery images for the cover
		if singleSceneURL != "" {
			sc.Covers = append(sc.Covers, sc.Gallery[0])
		}

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`table.video-description-list tbody`, func(id int, e *colly.HTMLElement) {
			// Cast
			e.ForEach(`tr:nth-child(1) a`, func(id int, e *colly.HTMLElement) {
				if strings.TrimSpace(e.Text) != "" {
					sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
					sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
				}
			})

			// Tags
			e.ForEach(`tr:nth-child(2) a`, func(id int, e *colly.HTMLElement) {
				tag := strings.TrimSpace(e.Text)
				sc.Tags = append(sc.Tags, tag)
			})

			// Date
			tmpDate, _ := goment.New(strings.TrimSpace(e.ChildText(`tr:nth-child(3) td:last-child`)), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Synposis
		sc.Synopsis = strings.TrimSpace(e.ChildText("div.accordion-body"))

		// Title
		sc.Title = e.ChildText("h1.page-title")

		// Scene ID -- Uses the ending number of the video url instead of the ID used for the directory that the video link is stored in(Maintains backwards compatibility with old scenes)
		tmpUrlStr, _ := strings.CutSuffix(e.Request.URL.String(), "/")
		tmp := strings.Split(tmpUrlStr, "/")
		siteIDstr := strings.Split(tmp[len(tmp)-1], "-")
		sc.SiteID = siteIDstr[len(siteIDstr)-1]

		if sc.SiteID != "" {
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

			// save only if we got a SceneID
			out <- sc
		}
	})

	siteCollector.OnHTML(`a.page-link[aria-label="Next"]:not(.disabled)`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.video-grid-view a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://kinkvr.com/videos/page1")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("kinkvr", "KinkVR", "https://static.rlcontent.com/shared/KINK/skins/web-10/branding/favicon.png", "kinkvr.com", KinkVR)
}
