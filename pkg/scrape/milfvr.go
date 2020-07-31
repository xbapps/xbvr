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

func MilfVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "milfvr"
	siteID := "MilfVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.milfvr.com")
	siteCollector := createCollector("www.milfvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Wankz"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`h1.detail__title`, func(id int, e *colly.HTMLElement) {
			sc.Title = e.Text
		})

		// Date
		e.ForEach(`div.detail__date_time span.detail__date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "DD MMMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.detail__date_time span.time`, func(id int, e *colly.HTMLElement) {
			if id == 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Text, "minutes", "", -1)))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		// Filenames
		base := e.Request.URL.Path
		base = strings.Replace(base, "/", "", -1)
		base = strings.Replace(base, sc.SiteID, "", -1)
		sc.Filenames = append(sc.Filenames, "milfvr-"+base+"180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, "milfvr-"+base+"gearvr-180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, "milfvr-"+base+"smartphone-180_180x180_3dh_LR.mp4")

		// Cover URLs
		tmpCover := "https://cdns-i.milfvr.com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/hero/large.jpg"
		sc.Covers = append(sc.Covers, tmpCover)

		// Gallery
		for _, x := range []string{"1", "2", "3", "4", "5", "6"} {
			tmpGallery := "https://cdns-i.milfvr.com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/thumbs/1280_" + x + ".jpg"
			sc.Gallery = append(sc.Gallery, tmpGallery)
		}

		// Synopsis
		e.ForEach(`div.detail__txt`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.tag-list__body a.tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast
		e.ForEach(`div.detail__models a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagenav__list a.pagenav__link`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`ul.cards-list a.card__video`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.milfvr.com/videos?o=d")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("milfvr", "MilfVR", "https://twivatar.glitch.me/milfvr", MilfVR)
}
