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

<<<<<<< HEAD
=======
<<<<<<< HEAD:pkg/scrape/tranzvr.go
func TranzVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "watranzvr"
	siteID := "TranzVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.tranzvr.com")
	siteCollector := createCollector("www.tranzvr.com")
=======
>>>>>>> master
func WankzVRSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.wankzvr.com", "www.milfvr.com")
	siteCollector := createCollector("www.wankzvr.com", "www.milfvr.com")
<<<<<<< HEAD
=======
>>>>>>> master:pkg/scrape/wankz.go
>>>>>>> master

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
<<<<<<< HEAD
		sc.Studio = "Wankz"
=======
		sc.Studio = "TranzVR"
>>>>>>> master
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
<<<<<<< HEAD
=======
<<<<<<< HEAD:pkg/scrape/tranzvr.go
		base = strings.Replace(base, sc.SiteID, "", -1)
		sc.Filenames = append(sc.Filenames, "tranzvr-"+base+"180_180x180_3dh_LR.mp4")

		// Cover URLs
		for _, x := range []string{"cover", "hero"} {
			tmpCover := "https://images.tranzvr.com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/550/" + x + ".webp"
=======
>>>>>>> master
		sc.Filenames = append(sc.Filenames, base+"180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"gearvr-180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"smartphone-180_180x180_3dh_LR.mp4")

		// Cover URLs
		for _, x := range []string{"cover", "hero"} {
			if scraperID == "milfvr" && x == "cover" {
				continue // MilfVR does not have a "cover" image unlike WankzVR
			}
			tmpCover := "https://cdns-i." + scraperID + ".com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/" + x + "/large.jpg"
<<<<<<< HEAD
=======
>>>>>>> master:pkg/scrape/wankz.go
>>>>>>> master
			sc.Covers = append(sc.Covers, tmpCover)
		}

		// Gallery
<<<<<<< HEAD
=======
<<<<<<< HEAD:pkg/scrape/tranzvr.go
		for _, x := range []string{"1"} {
			tmpGallery := "https://images.tranzvr.com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/thumbs/1024_" + x + ".jpg"
=======
>>>>>>> master
		size := "1024"
		if scraperID == "milfvr" {
			size = "1280"
		}
		for _, x := range []string{"1", "2", "3", "4", "5", "6"} {
			tmpGallery := "https://cdns-i." + scraperID + ".com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/thumbs/" + size + "_" + x + ".jpg"
<<<<<<< HEAD
=======
>>>>>>> master:pkg/scrape/wankz.go
>>>>>>> master
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
		if scraperID == "milfvr" {
			sc.Tags = append(sc.Tags, "milf")
		}

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

<<<<<<< HEAD
	siteCollector.Visit(URL + "videos?o=d")
=======
<<<<<<< HEAD:pkg/scrape/tranzvr.go
	siteCollector.Visit("https://www.tranzvr.com/videos?o=d")
=======
	siteCollector.Visit(URL + "videos?o=d")
>>>>>>> master:pkg/scrape/wankz.go
>>>>>>> master

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func WankzVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return WankzVRSite(wg, updateSite, knownScenes, out, "wankzvr", "WankzVR", "https://www.wankzvr.com/")
}

func MilfVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return WankzVRSite(wg, updateSite, knownScenes, out, "milfvr", "MilfVR", "https://www.milfvr.com/")
}

func init() {
<<<<<<< HEAD
	registerScraper("wankzvr", "WankzVR", "https://twivatar.glitch.me/wankzvr", WankzVR)
	registerScraper("milfvr", "MilfVR", "https://twivatar.glitch.me/milfvr", MilfVR)
=======
<<<<<<< HEAD:pkg/scrape/tranzvr.go
	registerScraper("tranzvr", "TranzVR", "https://twivatar.glitch.me/tranzvr", TranzVR)
=======
	registerScraper("wankzvr", "WankzVR", "https://twivatar.glitch.me/wankzvr", WankzVR)
	registerScraper("milfvr", "MilfVR", "https://twivatar.glitch.me/milfvr", MilfVR)
>>>>>>> master:pkg/scrape/wankz.go
>>>>>>> master
}
