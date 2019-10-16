package scrape

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
	"mvdan.cc/xurls/v2"
)

func VRLatina(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	logScrapeStart("vrlatina", "VRLatina")

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("vrlatina.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("vrlatina.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRLatina"
		sc.Site = "VRLatina"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Title
		e.ForEach(`div.video-info-left h2`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Covers
		e.ForEach(`script`, func(id int, e *colly.HTMLElement) {
			if strings.Contains(e.Text, "vidcontainer1") {
				url := xurls.Strict().FindAllString(e.Text, -1)[0]
				sc.Covers = append(sc.Covers, url)
			}
		})

		// Gallery
		e.ForEach(`div.sub-video a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("href"))
		})

		// Cast
		e.ForEach(`div.video-info-left h3 a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Tags
		e.ForEach(`div.video-tag-section a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if !funk.ContainsString(sc.Cast, tag) {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Synposis
		e.ForEach(`div.video-bottom-txt`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Release date / Duration
		e.ForEach(`div.video-info-left-icon span`, func(id int, e *colly.HTMLElement) {
			if id == 2 {
				tmpDate, _ := goment.New(strings.TrimSpace(e.Text), "DDMMM,YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
			if id == 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Text, "min", "", -1)))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// Scene ID
		e.ForEach(`link[rel=shortlink]`, func(id int, e *colly.HTMLElement) {
			sc.SiteID = strings.Split(e.Attr("href"), "?p=")[1]
			sc.SceneID = fmt.Sprintf("vrlatina-%v", sc.SiteID)
		})

		out <- sc
	})

	siteCollector.OnHTML(`span.pagination-wrap-inn a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.video-info-left h2 a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://vrlatina.com/videos/?typ=newest")

	if updateSite {
		updateSiteLastUpdate("vrlatina")
	}
	logScrapeFinished("vrlatina", "VRLatina")
	return nil
}

func init() {
	registerScraper("vrlatina", "VRLatina", VRLatina)
}
