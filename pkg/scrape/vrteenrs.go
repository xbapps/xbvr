package scrape

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRTeenrs(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vrteenrs"
	siteID := "VRTeenrs"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrteenrs.com", "www.vrteenrs.com")

	sceneCollector.OnHTML(`.list_item`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "International Media Company BV"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		sc.Title = e.ChildText(".title")
		coverURL := strings.TrimPrefix(e.ChildAttr("video", "poster"), "/")
		if coverURL == "" {
			coverURL = e.ChildAttr(".thumb img", "src")
		}
		if coverURL != "" {
			sc.Covers = append(sc.Covers, coverURL)

			// Scene ID - get from cover image URL
			re := regexp.MustCompile(`(?m)vrporn(\d+)`)
			sc.SiteID = re.FindStringSubmatch(coverURL)[1]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		}

		sc.Synopsis = e.ChildText(`.info .description`)

		e.ForEach(`.info .subtext`, func(id int, e *colly.HTMLElement) {
			tmp := strings.Split(e.Text, "Runtime: ")
			if len(tmp) > 1 {
				tmpDuration, err := strconv.Atoi(strings.Split(tmp[1], ":")[0])
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		if sc.Title != "" {
			out <- sc
		}
	})

	sceneCollector.Visit("https://www.vrteenrs.com/vrporn.php")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrteenrs", "VRTeenrs", "https://mcdn.vrporn.com/files/20170702081351/vrteenrs-icon-vr-porn-studio-vrporn.com-virtual-reality.png", VRTeenrs)
}
