package scrape

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func TmwVRnet(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	scraperID := "tmwvrnet"
	siteID := "TmwVRnet"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("tmwvrnet.com")
	siteCollector := createCollector("tmwvrnet.com")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "TeenMegaWorld"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Date & Duration
		e.ForEach(`.video-info-data`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.ChildText(`.video-info-date`), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.ChildText(`.video-info-time`), " min", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Title / Cover / ID
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Attr("title"))

			tmpCover := e.Request.AbsoluteURL(e.Attr("poster"))
			sc.Covers = append(sc.Covers, tmpCover)

			tmp := strings.Split(tmpCover, "/")
			sc.SiteID = tmp[5]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID + "-" + sc.Released
		})

		// Gallery
		e.ForEach(`div.photo-list img`, func(id int, e *colly.HTMLElement) {
			galleryURL := e.Request.AbsoluteURL(e.Attr("src"))
			if galleryURL == "" || galleryURL == "https://tmwvrnet.com/assets/vr/public/tour1/images/th5.jpg" {
				return
			}
			srcset := strings.Split(e.Attr("srcset"), ",")
			lastSrc := srcset[len(srcset)-1]
			if lastSrc != "" {
				galleryURL = e.Request.AbsoluteURL(strings.TrimSpace(strings.Split(lastSrc, " ")[0]))
			}
			sc.Gallery = append(sc.Gallery, galleryURL)
		})

		// Synopsis
		e.ForEach(`p.video-description-text`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.video-tag-list a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`.video-actor-list a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Filenames
		// NOTE: no way to guess filename

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination__element.next a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.thumb-photo`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))

		if strings.Contains(sceneURL, "trailers") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {

				sceneCollector.Visit(sceneURL)
			}
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://tmwvrnet.com/categories/movies.html")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("tmwvrnet", "TmwVRnet", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/26/logo_crop_1623330575.png", "tmwvrnet.com", TmwVRnet)
}
