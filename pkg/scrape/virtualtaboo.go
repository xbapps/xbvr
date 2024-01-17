package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VirtualTaboo(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "virtualtaboo"
	siteID := "VirtualTaboo"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("virtualtaboo.com")
	siteCollector := createCollector("virtualtaboo.com")

	durationRegEx := regexp.MustCompile(`(?:(\d+) hour(?:s)? )?(\d+) min`)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VirtualTaboo"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		e.ForEach(`#player`, func(id int, e *colly.HTMLElement) {
			sc.SiteID = strings.Split(e.Attr("data-poster-index"), ":")[0]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`div.video-detail h1`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Filenames
		base := strings.Split(e.Request.URL.Path, "/")[2]
		sc.Filenames = append(sc.Filenames, base+"-files-smartphone_180_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-gear_180_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-psvr_180_sbs.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-oculus_180_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-oculus5k_180_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-oculus5k10_180_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-6k_180_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"-files-7k_180_LR.mp4")

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`div.gallery-item:not(.link) a.gallery-image`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(strings.Split(e.Attr("href"), "?")[0]))
		})

		// Synopsis
		e.ForEach(`div.video-detail .description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.tag-list a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// trailer details
		sc.TrailerType = "load_json"
		params := models.TrailerScrape{SceneUrl: `https://virtualtaboo.com/gizmo/videoinfo/` + sc.SiteID, RecordPath: "sources", ContentPath: "url", QualityPath: "title"}
		strParma, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParma)

		// Cast
		e.ForEach(`div.video-detail .info a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date & Duration
		e.ForEach(`div.right-info div.info`, func(id int, e *colly.HTMLElement) {
			tmpData := funk.ReverseStrings(strings.Split(e.Text, "\n"))

			tmpDate, _ := goment.New(strings.TrimSpace(tmpData[1]), "DD MMMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")

			tmpDuration := 0
			durationMatch := durationRegEx.FindStringSubmatch(tmpData[3])
			if len(durationMatch) == 2 {
				tmpDuration, _ = strconv.Atoi(durationMatch[1])
			} else if len(durationMatch) == 3 {
				hours, _ := strconv.Atoi(durationMatch[1])
				minutes, _ := strconv.Atoi(durationMatch[2])
				tmpDuration = hours*60 + minutes
			}
			sc.Duration = tmpDuration
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.video-card__item a[class=image-container]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://virtualtaboo.com/videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("virtualtaboo", "VirtualTaboo", "https://static-src.virtualtaboo.com/img/mobile-logo.png", "virtualtaboo.com", VirtualTaboo)
}
