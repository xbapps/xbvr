package scrape

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func DarkRoomVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "darkroomvr"
	siteID := "DarkRoomVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("darkroomvr.com")
	siteCollector := createCollector("darkroomvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VirtualTaboo"
		sc.Site = siteID
		sc.SiteID = ""
		sc.HomepageURL = e.Request.URL.String()

		// Title
		e.ForEach(`h1.video-detail__title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`div.video-detail__image-container img`, func(id int, e *colly.HTMLElement) {
			coverUrl := e.Attr("src")
			sc.Covers = append(sc.Covers, coverUrl)
		})

		// Gallery
		e.ForEach(`div.video-detail__gallery a.image-container`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("href"))
		})

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.video-detail__desktop-sidebar div.video-info__text a`, func(id int, e *colly.HTMLElement) {
			if strings.TrimSpace(e.Text) != "" {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
				sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Attr("href")}
			}
		})

		// Tags
		e.ForEach(`div.tags__container a.tags__item`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, strings.ToLower(tag))
			}
		})

		// Synposis
		e.ForEach(`.video-detail__description .hidden`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(e.Text), "Read less"))
		})

		// Release date / Duration
		e.ForEach(`div.video-info__time`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, "•")
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(parts[0], "MIN", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
			tmpDate, _ := goment.New(strings.TrimSpace(parts[1]), "DD MMMM, YYYY:")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Scene ID
		e.ForEach(`a[href*="signup.php"]`, func(id int, e *colly.HTMLElement) {
			url := e.Attr("href")
			if strings.Contains(url, "vid=") {
				sc.SiteID = url[strings.LastIndex(url, "=")+1:]
			}
		})
		// Filenames (only a guess for now, according to the sample files)
		suffixes := []string{"4k", "5k", "5k10", "6k", "7k", "960p", "1440p", "psvr_1440p"}
		base := e.Request.URL.Path
		base = strings.TrimPrefix(base, "/video/")
		for _, suffix := range suffixes {
			sc.Filenames = append(sc.Filenames, "drvr-"+base+"-"+suffix+"_180_LR.mp4")
		}
		release := strings.TrimSuffix(e.ChildAttr(`meta[property="og:video"]`, "content"), "-ws_4k.mp4")
		relname := release[strings.LastIndex(release, "/")+1:]
		for _, suffix := range suffixes {
			sc.Filenames = append(sc.Filenames, relname+"-"+suffix+".mp4")
		}

		// trailer details
		sc.TrailerType = "load_json"
		params := models.TrailerScrape{SceneUrl: `https://darkroomvr.com/api/vrplayer/video/detail/` + sc.SiteID, RecordPath: "sources", ContentPath: "url", QualityPath: "title"}
		strParma, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParma)

		if sc.SiteID != "" {
			sc.SceneID = fmt.Sprintf("darkroomvr-%v", sc.SiteID)

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
		siteCollector.Visit("https://darkroomvr.com/video/")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("darkroomvr", "DarkRoomVR", "https://static.darkroomvr.com/img/favicon/apple-touch-180.png", "darkroomvr.com", DarkRoomVR)
}
