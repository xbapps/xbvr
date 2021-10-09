package scrape

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func GroobyVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "groobyvr"
	siteID := "GroobyVR"
	allowedDomains := []string{"groobyvr.com", "www.groobyvr.com"}
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector(allowedDomains...)
	siteCollector := createCollector(allowedDomains...)
	vodCollector := createCollector(allowedDomains...)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "GroobyVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Title
		sc.Title = strings.Replace(e.ChildText(`title`), "Grooby VR: ", "", -1)

		// Cast
		e.ForEach(`div.trailer_toptitle_left a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Cover URL
		coverURL := "https:" + e.Request.AbsoluteURL(e.ChildAttr("div.player-thumb img", "src"))
		sc.Covers = append(sc.Covers, coverURL)

		// Scene ID - get from URL
		tmps := strings.Split(coverURL, "/")
		tmp := strings.Replace(tmps[len(tmps)-1], ".jpg", "", -1)
		sc.SiteID = tmp
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.trailerblock p`))

		// Date
		dateString := strings.Replace(e.ChildText(`div.set_meta`), "Added ", "", -1)
		tmpDate, _ := goment.New(dateString, "MMMM D, YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		// Pull data from vod page - not every scene has a vod link
		ctx := colly.NewContext()
		ctx.Put("scene", &sc)
		vodURL := e.ChildAttr("a.downBtnbuy", "href")
		if !strings.Contains(vodURL, "/tour") {
			vodCollector.Request("GET", vodURL, nil, ctx, nil)
		}

		out <- sc
	})

	vodCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(*models.ScrapedScene)

		// Gallery
		sc.Gallery = e.ChildAttrs("div.gallery-group img", "data-orig-file")

		// Duration - not every vod page has a duration listed
		r := strings.NewReplacer("Length: ", "", "Length ", "")
		tmpDuration := r.Replace(e.ChildText(`ul li:contains(length)`))
		tmpDuration = strings.Split(tmpDuration, "m")[0]
		duration, err := strconv.Atoi(tmpDuration)
		if err == nil && duration > 0 {
			sc.Duration = duration
		}

		// Tags
		e.ForEach(`span.meta-tag a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Not every page has tags so use categories as well
		e.ForEach(`span.meta-cat a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

	})

	siteCollector.OnHTML(`div.videohere a`, func(e *colly.HTMLElement) {
		sceneURL := "https:" + e.Request.AbsoluteURL(e.Attr("href"))

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`div.pagination li a:not(.active)`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.Visit("https://www.groobyvr.com/tour/categories/movies/1/latest/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("groobyvr", "GroobyVR", "https://pbs.twimg.com/profile_images/981677396695773184/-kKaWumY_200x200.jpg", GroobyVR)
}
