package scrape

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func POVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, company string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("povr.com")
	siteCollector := createCollector("povr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(scraperID) + "-" + sc.SiteID

		// Title
		e.ForEach(`h1.player__title`, func(id int, e *colly.HTMLElement) {
			sc.Title = e.Text
		})

		// Date
		e.ForEach(`div.player__meta div.player__meta-item`, func(id int, e *colly.HTMLElement) {

			if id == 2 {
				tmpDate, _ := goment.New(e.Text, "DD MMMM, YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
			// Duration
			//		e.ForEach(`div.detail__date_time span.time`, func(id int, e *colly.HTMLElement) {
			if id == 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Text, "min,", "", -1)))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// Filenames
		base := e.Request.URL.Path
		base = strings.Split(strings.Replace(base, "/", "-", -1), sc.SiteID)[0]
		base = strings.TrimPrefix(base, "-")
		sc.Filenames = append(sc.Filenames, base+"180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"gearvr-180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"smartphone-180_180x180_3dh_LR.mp4")

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Gallery
		//		size := "1024"
		//		for _, x := range []string{"1", "2", "3", "4", "5", "6"} {
		//			tmpGallery := "https://cdns-i." + scraperID + ".com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/thumbs/" + size + "_" + x + ".jpg"
		//			sc.Gallery = append(sc.Gallery, tmpGallery)
		//		}

		// Synopsis
		e.ForEach(`p.player-link__description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`a.player-link__tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast
		e.ForEach(`a.player-link__actor`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		out <- sc
	})

	//	siteCollector.OnHTML(`div.teaser-video a.teaser-video__main`, func(e *colly.HTMLElement) {
	//		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
	//		siteCollector.Visit(pageURL)
	//	})

	siteCollector.OnHTML(`div.teaser-video a.teaser-video__main`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://povr.com/" + scraperID)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addPOVRScraper(id string, name string, company string, avatarURL string) {
	suffixedName := name
	if company != "POVR.COM" {
		suffixedName += " (POVR)"
	}
	registerScraper(id, suffixedName, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return POVR(wg, updateSite, knownScenes, out, id, name, company)
	})
}

func init() {
	addPOVRScraper("povr-originals", "POVR Originals", "POVR.COM", "https://images.povr.com/img/povr/android-icon-192x192.png")
	addPOVRScraper("herpovr", "herPOVR", "POVR.COM", "https://images.povr.com/img/povr/android-icon-192x192.png")
}
