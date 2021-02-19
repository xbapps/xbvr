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

func TmwVRnet(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "tmwvrnet"
	siteID := "TmwVRnet"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("tmwvrnet.com")
	siteCollector := createCollector("tmwvrnet.com")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "TeenMegaWorld"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Date
		e.ForEach(`li.icons-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Title / Cover / ID
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Attr("title"))

			tmpCover := e.Attr("poster")
			sc.Covers = append(sc.Covers, tmpCover)

			tmp := strings.Split(tmpCover, "/")
			sc.SiteID = tmp[5]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID + "-" + sc.Released
		})

		// Gallery
		e.ForEach(`ul.slider-set img`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				base := e.Request.AbsoluteURL(e.Attr("src"))
				base = strings.Split(base, "?")[0]
				sc.Gallery = append(sc.Gallery, base)
			}
		})

		// Synopsis
		e.ForEach(`p.ep-desc`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.Replace(strings.Replace(strings.TrimSpace(e.Text), "Read more", "", -1), "\n", "", -1)
		})

		// Tags
		e.ForEach(`p.ep-tags a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Cast
		e.ForEach(`div.ep-info-l p a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		// Duration
		e.ForEach(`li.icons-length`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Text, " min", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames
		// NOTE: no way to guess filename

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination a.in_stditem`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.videos-container div.videos-item a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if strings.Contains(sceneURL, "trailers") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				sceneCollector.Visit(sceneURL)
			}
		}
	})

	siteCollector.Visit("https://tmwvrnet.com/categories/movies.html")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("tmwvrnet", "TmwVRnet", "https://pbs.twimg.com/profile_images/832208391967797250/1rEowkN6_200x200.jpg", TmwVRnet)
}
