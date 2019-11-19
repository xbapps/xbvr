package scrape

import (
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func SexBabesVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	logScrapeStart("sexbabesvr", "SexBabesVR")

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("sexbabesvr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("sexbabesvr.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "SexBabesVR"
		sc.Site = "SexBabesVR"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		tmp2 := strings.Split(tmp[len(tmp)-1], "-")[0]
		sc.SiteID = strings.Replace(tmp2, "vrh", "", -1)
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`h1.title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`div.splash-screen`, func(id int, e *colly.HTMLElement) {
			base := e.Attr("style")
			base = strings.Split(base, "background-image: url(")[1]
			base = strings.Split(base, ");")[0]
			base = strings.Split(base, "?")[0]
			sc.Covers = append(sc.Covers, base)
		})

		// Gallery
		e.ForEach(`figure[itemprop=associatedMedia] a`, func(id int, e *colly.HTMLElement) {
			base := e.Request.AbsoluteURL(e.Attr("href"))
			base = strings.Split(base, "?")[0]
			sc.Gallery = append(sc.Gallery, base)
		})

		// Synopsis
		e.ForEach(`.video-group-bottom`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`.video-tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Cast
		e.ForEach(`div.video-actress-name a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`div.video-info span.date-display-single`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.video-additional div.grid-one-time span`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.DOM.Parent().Text(), "min.", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames
		e.ForEach(`div.modal a.vd-row`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("href"))
			base := origURL.Query().Get("response-content-disposition")
			base = strings.Replace(base, "attachment; filename=", "", -1)
			base = strings.Replace(base, "\"", "", -1)
			base = strings.Replace(base, "_trailer", "", -1)
			if !funk.ContainsString(sc.Filenames, base) {
				sc.Filenames = append(sc.Filenames, base)
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pager li.is-active a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.cf div.grid-one-hov-field a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://sexbabesvr.com/virtualreality/list")

	if updateSite {
		updateSiteLastUpdate("sexbabesvr")
	}
	logScrapeFinished("sexbabesvr", "SexBabesVR")
	return nil
}

func init() {
	registerScraper("sexbabesvr", "SexBabesVR", "https://twivatar.glitch.me/sexbabes_vr", SexBabesVR)
}
