package scrape

import (
	"strings"
	"sync"
	"strconv"
	"fmt"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/tidwall/gjson"
)

// func isGoodTag(lookup string) bool {
// 	switch lookup {
// 	case
// 		"VR",
// 		"Sex":
// 		return false
// 	}
// 	return true
// }


func BrazzersVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "brazzersvr"
	siteID := "BrazzersVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.brazzersvr.com")
	siteCollector := createCollector("www.brazzersvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = siteID
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]


		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		e.ForEach(`script[type="application/ld+json"]`, func(id int, e *colly.HTMLElement) {
			// Date - This currently returns all the same date for every scene, will have to check to see if it gets more accurate
			tmpDate, _ := goment.New(gjson.Get(e.Text, "uploadDate").String(), "YYYY-MM-DD")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
			
			// Cover
			sc.Covers = append(sc.Covers, gjson.Get(e.Text, "thumbnailUrl").String())
			
			// Synopsis
			sc.Synopsis = strings.TrimSpace(gjson.Get(e.Text, "description").String())

			// Title
			sc.Title = gjson.Get(e.Text, "name").String()
		})

		// Tags
		e.ForEach(`div.sc-vdkjux-1 a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})


		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.sc-1b6bgon-5 span a`, func(id int, e *colly.HTMLElement) {
			name := strings.TrimSpace(e.Text)
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{
				Source:     scraperID + " scrape",
				ProfileUrl: e.Request.AbsoluteURL(e.Attr("href")),
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`body`, func(e *colly.HTMLElement) {
		e.ForEachWithBreak(`a.e1qkfw3j3`, func(id int, e *colly.HTMLElement) bool {
			tmp := strings.Split(e.Request.URL.String(), "/")
			currentPage, _ := strconv.Atoi(tmp[len(tmp)-1])
			if !limitScraping {
				siteCollector.Visit(fmt.Sprintf("https://www.brazzersvr.com/videos/sortby/releasedate/page/%d", currentPage + 1))
			}
			return false
		})
	})

	siteCollector.OnHTML(`a.e1qkfw3j3`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))
		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.brazzersvr.com/videos/sortby/releasedate")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("brazzersvr", "BrazzersVR", "https://images-assets-ht.project1content.com/BrazzersVR/Common/Favicon/63e2a8fdbdbe16.78976344.jpg", "brazzersvr.com", BrazzersVR)
}
