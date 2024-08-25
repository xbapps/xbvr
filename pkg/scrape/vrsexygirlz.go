package scrape

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRSexygirlz(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()

	scraperID := "vrsexygirlz"
	siteID := "VRSexyGirlz"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrsexygirlz.com", "www.vrsexygirlz.com")
	siteCollector := createCollector("vrsexygirlz.com", "www.vrsexygirlz.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRSexyGirlz"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Title
		e.ForEach(`div.content-block div.ep-info-l h2`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Gallery
		e.ForEach(`div.bx-set-pager img`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("src")))
		})

		// Cover URLs
		if len(sc.Gallery) > 0 {
			sc.Covers = append(sc.Covers, sc.Gallery[0])
		}

		// Synopsis
		e.ForEach(`div.episode-description div.ep-desc`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Cast
		e.ForEach(`div.ep-info-model a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, e.Text)

		})

		// Date
		e.ForEach(`ul.ep-info-r li.icons-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`ul.ep-info-r li.icons-length`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.Split(e.Text, ":")[0])
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Tags
		// no tags on this site

		// Filenames
		// NOTE: no way to guess filename

		// Scene ID
		e.ForEach(`link[rel="shortlink"]`, func(id int, e *colly.HTMLElement) {
			url := e.Attr("href")
			sc.SiteID = url[strings.LastIndex(url, "=")+1:]
		})

		if sc.SiteID != "" {
			sc.SceneID = fmt.Sprintf("vrsexygirlz-%v", sc.SiteID)

			// save only if we got a SceneID
			out <- sc
		}
	})

	siteCollector.OnHTML(`div.wpx-pagination a.next`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.post-content div.episode div.episode-info div.ep-info-l > a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.vrsexygirlz.com")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrsexygirlz", "VRSexyGirlz", "https://www.vrsexygirlz.com/wp-content/uploads/2017/03/FV.png", "vrsexygirlz.com", VRSexygirlz)
}
