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
	scraperID := "vrlatina"
	siteID := "VRLatina"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrlatina.com")
	siteCollector := createCollector("vrlatina.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRLatina"
		sc.Site = siteID
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
			if strings.TrimSpace(e.Text) != "" {
				sc.Cast = append(sc.Cast, strings.TrimSpace(strings.ReplaceAll(e.Text, "!", "")))
			}
		})

		// Tags
		// Note: rare multi-girl scenes only feature one cast name, other girls are added as tags only
		edgecases := map[string]bool{
			"Alicia Feliz":     true,
			"Karina Rojo":      true,
			"Samantha Sanchez": true,
			"Diana Dimon":      true,
		}

		e.ForEach(`div.video-tag-section a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)

			// Check tag for edgecases and add missing names to sc.Cast
			if edgecases[tag] {
				if !funk.Contains(sc.Cast, tag) {
					sc.Cast = append(sc.Cast, tag)
				}
			}

			// Note: need case-insensitive compare here, castname-tags can be lower case (nikol sparta)
			tagIsActor := false
			for _, actor := range sc.Cast {
				if strings.EqualFold(actor, tag) {
					tagIsActor = true
				}
			}

			if !tagIsActor {
				sc.Tags = append(sc.Tags, strings.ToLower(tag))
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
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrlatina", "VRLatina", "https://pbs.twimg.com/profile_images/979329978750898176/074YPl3H_200x200.jpg", VRLatina)
}
