package scrape

import (
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

		// Gallery URLs
		// This example is using colly.HTMLElement.ChildAttrs which returns
		// the stripped text content of all the matching element's attributes.
		//
		// For example, given the following HTML:
		//   <div class="gallery">
		//     <img src="http://example.com/image01.png">
		//     <img src="http://example.com/image02.png">
		//     <img src="http://example.com/image03.png">
		//
		// e.ChildAttrs(".gallery img", "src") returns:
		//   [ "http://example.com/image01.png",
		//     "http://example.com/image02.png",
		//     "http://example.com/image03.png" ]
		//sc.Gallery = e.ChildAttrs(".gallery img", "src")

		// Tags
		// This example is using colly.HTMLElement.ForEach which iterates
		// over the elements matched by the first argument and calls the
		// callback function on every HTMLElement match.
		//
		// For example, given the following HTML:
		//   <div class="tags">
		//     <ul>
		//       <li>tag1</li>
		//       <li>tag2</li>
		//       <li>tag3</li>
		//     </ul>
		//
		// e.Text returns a tag on every iteration
		//e.ForEach(`.tags ul li`, func(id int, e *colly.HTMLElement) {
		//	sc.Tags = append(sc.Tags, e.Text)
		//})

		// Duration
		//
		// We only care about the minutes. You'll need to parse the string if it's
		// in HH:MM:SS or MM:SS format.
		//
		// Assuming: <span class="runtime">32:00</span>
		//tmpDuration, err := strconv.Atoi(strings.Split(e.ChildText(`span.runtime`), ":")[0])
		//if err == nil {
		//	sc.Duration = tmpDuration
		//}

		out <- sc
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
	registerScraper("groobyvr", "GroobyVR", "https://twivatar.glitch.me/groobyvr", GroobyVR)
}
