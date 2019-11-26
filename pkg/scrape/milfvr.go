package scrape

import (
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
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
		e.ForEach(`div.videoDetails h4`, func(id int, e *colly.HTMLElement) {
			sc.Title = e.Text
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
			tmpGallery := "https://cdns-i.milfvr.com/" + sc.SiteID[0:1] + "/" + sc.SiteID[0:4] + "/" + sc.SiteID + "/thumbs/700_" + x + ".jpg"
			sc.Gallery = append(sc.Gallery, tmpGallery)
		}

		// Synopsis
		e.ForEach(`div.videoDetails p`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Synopsis = strings.TrimSpace(e.Text)
			}
		})

		// Cast / Duration / Uploaded / Tags
		e.ForEach(`div.videoDetails ul.videoInfo li`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				e.DOM.Find(`a`).Each(func(id int, e *goquery.Selection) {
					sc.Cast = append(sc.Cast, e.Text())
				})
			}
			if id == 1 {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(strings.Replace(e.Text, "min", "", -1), "Time:", "", -1)))
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
			if id == 2 {
				tmpDate, _ := goment.New(strings.Replace(e.Text, "Uploaded:", "", -1), "DD MMM, YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
			if id == 3 {
				e.DOM.Find(`a`).Each(func(id int, e *goquery.Selection) {
					sc.Tags = append(sc.Tags, e.Text())
				})
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pager a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.milfVideos div.videoCover a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://www.milfvr.com/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("milfvr", "MilfVR", "https://twivatar.glitch.me/milfvr", MilfVR)
}
