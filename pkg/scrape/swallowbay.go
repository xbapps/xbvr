package scrape

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func SwallowBay(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "swallowbay"
	siteID := "SwallowBay"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("swallowbay.com")
	siteCollector := createCollector("swallowbay.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "SwallowBay"
		sc.Site = siteID
		sc.SiteID = ""
		sc.HomepageURL = e.Request.URL.String()

		regexpSceneID := regexp.MustCompile(`\-(\d+)\.html$`)
		sc.SiteID = regexpSceneID.FindStringSubmatch(e.Request.URL.Path)[1]

		// Title
		e.ForEach(`div.content-title h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			coverUrl := e.Attr("poster")
			sc.Covers = append(sc.Covers, coverUrl)
		})

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.content-models a`, func(id int, e *colly.HTMLElement) {
			if strings.TrimSpace(e.Text) != "" {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Attr("title")))
				sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Attr("href")}
			}
		})

		// Tags
		ignoreTags := []string{"vr 180", "vr 6k", "8k", "iphone", "ultra high definition"}
		e.ForEach(`div.content-tags a`, func(id int, e *colly.HTMLElement) {
			tag := strings.ToLower(strings.TrimSpace(e.Text))
			if tag != "" {
				for _, v := range ignoreTags {
					if tag == v {
						return
					}
				}
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Synposis
		e.ForEach(`div.content-desc.active`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(strings.TrimSpace(e.Text))
		})

		// Release date
		e.ForEach(`div.content-data div.content-date`, func(id int, e *colly.HTMLElement) {
			date := strings.TrimSuffix(e.Text, "Date: ")
			tmpDate, _ := goment.New(strings.TrimSpace(date), "Do MMM, YYYY:")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.content-data div.content-time`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, ":")
			if len(parts) > 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// No filename information yet

		sc.TrailerType = "urls"
		var trailers []models.VideoSource
		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			trailers = append(trailers, models.VideoSource{URL: e.Attr("src"), Quality: strings.TrimSpace(e.Attr("quality"))})
		})
		trailerJson, _ := json.Marshal(models.VideoSourceResponse{VideoSources: trailers})
		sc.TrailerSrc = string(trailerJson)

		if sc.SiteID != "" {
			sc.SceneID = fmt.Sprintf("swallowbay-%v", sc.SiteID)

			// save only if we got a SceneID
			out <- sc
		}
	})

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.-video div.item-name a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://swallowbay.com/most-recent/")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("swallowbay", "SwallowBay", "https://swallowbay.com/templates/swallowbay/images/favicons/apple-icon-180x180.png", "swallowbay.com", SwallowBay)
}
