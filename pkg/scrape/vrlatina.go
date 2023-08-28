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

func VRLatina(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	scraperID := "vrlatina"
	siteID := "VRLatina"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrlatina.com")
	siteCollector := createCollector("vrlatina.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRLatina"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Title
		e.ForEach(`div.content-title h2`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Covers
		coverurl := e.ChildAttr(`meta[property="og:image"]`, "content")
		if coverurl != "" {
			sc.Covers = append(sc.Covers, coverurl)
		}

		// Gallery
		e.ForEach(`div.video-gallery a.video-gallery-item`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("href"))
		})

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.content-links.-models a`, func(id int, e *colly.HTMLElement) {
			if strings.TrimSpace(e.Text) != "" {
				sc.Cast = append(sc.Cast, strings.TrimSpace(strings.ReplaceAll(e.Text, "!", "")))
				sc.ActorDetails[e.Text] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Attr("href")}
			}
		})

		// Tags

		e.ForEach(`div.content-links.-tags a.tag`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, strings.ToLower(tag))
			}
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "deo-video source", ContentPath: "src", QualityPath: "quality", ContentBaseUrl: "https:"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Synposis
		e.ForEach(`div.content-desc`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Release date / Duration
		e.ForEach(`div.info-elem.-length span.sub-label`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				durationParts := strings.Split(strings.TrimSpace(e.Text), ":")
				hours, minutes := 0, 0
				if len(durationParts) == 2 {
					minutes, _ = strconv.Atoi(durationParts[0])
				} else if len(durationParts) == 3 {
					hours, _ = strconv.Atoi(durationParts[0])
					minutes, _ = strconv.Atoi(durationParts[1])
				}
				sc.Duration = hours*60 + minutes
			}
			if id == 1 {
				tmpDate, _ := goment.New(strings.TrimSpace(e.Text), "MMM DD, YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
		})

		// Scene ID
		url := e.ChildAttr(`link[rel="canonical"]`, "href")
		r := regexp.MustCompile(`-(\d+).html`)
		matches := r.FindStringSubmatch(url)
		if matches != nil {
			sc.SiteID = matches[1]
			sc.SceneID = fmt.Sprintf("vrlatina-%v", sc.SiteID)

			// save only if we got a SceneID
			out <- sc
		}
	})

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.item-col.-video a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://vrlatina.com/most-recent/")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrlatina", "VRLatina", "https://pbs.twimg.com/profile_images/979329978750898176/074YPl3H_200x200.jpg", "vrlatina.com", VRLatina)
}
