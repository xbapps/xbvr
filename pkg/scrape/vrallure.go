package scrape

import (
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRAllure(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vrallure"
	siteID := "VRAllure"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrallure.com")
	siteCollector := createCollector("vrallure.com")

	// Regex for original resolution of both covers and gallery
	reGetOriginal := regexp.MustCompile(`^(https?:\/\/b8h6h9v9\.ssl\.hwcdn\.net\/vra\/)(?:largethumbs|hugethumbs|rollover_large)(\/.+)-c\d{3,4}x\d{3,4}(\.\w{3,4})$`)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRAllure"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		tmp2 := strings.Split(tmp[len(tmp)-1], "_")[0]
		sc.SiteID = strings.Replace(tmp2, "vra", "", -1)
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Date
		e.ForEach(`div.scene-details p.publish-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Title / Cover
		e.ForEach(`deo-video`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Attr("title"))

			tmpParts := reGetOriginal.FindStringSubmatch(e.Request.AbsoluteURL(e.Attr("cover-image")))
			sc.Covers = append(sc.Covers, tmpParts[1]+"largethumbs"+tmpParts[2]+tmpParts[3])
		})

		// Gallery
		e.ForEach(`div.swiper-wrapper div.swiper-slide img`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				// Note: rollover_large version of gallery images on this site is very small and the website doesn't show any other resolutions itself.
				// However this CDN can resize on request but that's a bit slower. It's possible to use the huge version and resize to 1422x800 like this:
				// https://k3y8c8f9.ssl.hwcdn.net/vra/rollover_huge/vra0043_kaycarter_180/vra0043_kaycarter_180_01-1422x800.jpg
				tmpParts := reGetOriginal.FindStringSubmatch(e.Request.AbsoluteURL(e.Attr("src")))
				sc.Gallery = append(sc.Gallery, tmpParts[1]+"rollover_large"+tmpParts[2]+tmpParts[3])
			}
		})

		// Synopsis
		e.ForEach(`div.scene-details div.video-desc p.desc span`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"HD Virtual Reality Porn": true,
			"Virtual Reality Porn":    true,
			"VR Porn":                 true,
		}

		// Tags
		e.ForEach(`div.scene-details p.tag-container a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" && !skiptags[tag] {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// Cast
		e.ForEach(`div.scene-details p.model-name a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		// Duration
		// Note the element `div.scene-details p.duration` is there but currently inside an HTML comment block
		/*
			e.ForEach(`div.scene-details p.duration`, func(id int, e *colly.HTMLElement) {
				tmpDuration, err := strconv.Atoi(strings.TrimSpace(e.Text))
				if err == nil {
					sc.Duration = tmpDuration / 60
				}
			})
		*/

		// Filenames
		e.ForEach(`input.stream-input-box`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("value"))
			sc.Filenames = append(sc.Filenames, origURL.Query().Get("name"))
		})

		out <- sc
	})

	siteCollector.OnHTML(`ul.pagination li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.row h4.latest-scene-title a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://vrallure.com/scenes")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrallure", "VRAllure", "https://z5w6x5a4.ssl.hwcdn.net/sites/vra/favicon/apple-touch-icon-180x180.png", VRAllure)
}
