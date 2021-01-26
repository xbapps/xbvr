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
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.Studio = "GroobyVR"
		sc.Title = strings.TrimSpace(e.ChildText("span.settitle"))
		sc.Cast = append(sc.Cast, e.ChildText("span.modellink"))

		coverURL := e.Request.AbsoluteURL(e.ChildAttr("div.bigvideo div.videohere img", "src"))
		sc.Covers = append(sc.Covers, coverURL)

		tmps := strings.Split(coverURL, "/")
		tmp := strings.Replace(tmps[len(tmps)-1], ".jpg", "", -1)
		sc.SiteID = tmp
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.set_meta p:not([class])`))

		dateString := strings.Replace(e.ChildText(`p.set_meta_details`), "Added: ", "", -1)
		tmpDate, _ := goment.New(dateString, "Do MMMM YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		// not every scene has a vod link
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

		sc.Gallery = e.ChildAttrs("div.gallery-group img", "data-orig-file")

		// not every vod page has a duration listed
		tmpDuration := strings.Replace(e.ChildText(`ul li:contains(length)`), "Length: ", "", -1)
		tmpDuration = strings.Split(tmpDuration, "m")[0]
		duration, err := strconv.Atoi(tmpDuration)
		if err == nil && duration > 0 {
			sc.Duration = duration
		}
	})

	siteCollector.OnHTML(`div.frontpage_sexyvideo h4 a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`div.pagination li a:not(.active)`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.Visit("https://groobyvr.com/tour/categories/movies/1/latest/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("groobyvr", "GroobyVR", "https://pbs.twimg.com/profile_images/981677396695773184/-kKaWumY_200x200.jpg", GroobyVR)
}
