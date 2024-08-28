package scrape

import (
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VirtualPee(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "virtualpee"
	siteID := "VirtualPee"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("virtualpee.com")
	siteCollector := createCollector("virtualpee.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = siteID
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://virtualpee.com", "https://members.virtualpee.com", 1)

		// Scene ID - get from URL
		tmp := strings.SplitN(sc.HomepageURL, "-", 2)
		sc.SiteID = tmp[1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = e.Request.Ctx.Get("title")

		// Date
		tmpDate, _ := goment.New(e.Request.Ctx.Get("date"), "MMM DD, YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		// Duration
		tmpDuration, err := strconv.Atoi(strings.Split(e.ChildText(`li.vid_duration`), ":")[0])
		if err == nil {
			sc.Duration = tmpDuration
		}

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.col-md-12.right p`))

		// Tags
		e.ForEach(`h5.tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cover
		sc.Covers = append(sc.Covers, e.ChildAttr(`figure span.vid_wrap img`, "src"))

		// Gallery
		e.ForEach(`ul.bxslider_pics li a img.lazy`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("src"))
		})

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`h2.video_title strong a`, func(id int, e *colly.HTMLElement) {
			name := strings.TrimSpace(e.Text)
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{
				Source:     scraperID + " scrape",
				ProfileUrl: e.Request.AbsoluteURL(e.Attr("href")),
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`a.next_page`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`li.row`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`div.col-md-4.right h2 a`, "href"))

		ctx := colly.NewContext()
		ctx.Put("date", strings.TrimSpace(e.ChildText("figure figcaption ul li strong")))
		ctx.Put("title", strings.TrimSpace(e.ChildText("div.col-md-4.right h2 a")))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("date", "")

		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)

	} else {
		siteCollector.Visit("https://virtualpee.com/videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("virtualpee", "VirtualPee", "https://media.virtualpee.com/assets/images/logo.png", "virtualpee.com", VirtualPee)
}
