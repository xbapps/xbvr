package scrape

import (
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
)

func VRBangers(wg *sync.WaitGroup, knownScenes []string, out *[]ScrapedScene) error {
	siteCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
		colly.Async(true),
	)
	siteCollector.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: maxCollyThreads})

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("vrbangers.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
		colly.Async(true),
	)
	sceneCollector.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: maxCollyThreads})

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = "VRBangers"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		e.ForEach(`link[rel=shortlink]`, func(id int, e *colly.HTMLElement) {
			tmp := strings.Split(e.Attr("href"), "?p=")
			sc.SiteID = tmp[1]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`div.video-info-title h1 span`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = e.Text
			}
		})

		// Date
		e.ForEach(`p[itemprop=datePublished]`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "DD MMMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`p.minutes`, func(id int, e *colly.HTMLElement) {
			minutes := strings.Split(e.Text, ":")[0]
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(minutes))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames
		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				basePath := strings.Split(strings.Replace(e.Attr("src"), "//", "", -1), "/")
				baseName := strings.Replace(basePath[1], "vrb_", "", -1)

				filenames := []string{"6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

				for i := range filenames {
					filenames[i] = "VRBANGERS_" + baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		// Cover URLs
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("poster")))
		})

		e.ForEach(`img.girls_image`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("src")))
		})

		// Gallery
		e.ForEach(`div#single-video-gallery-free a,div.old-gallery a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		// Synopsis
		e.ForEach(`div.mainContent`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.video-tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast
		e.ForEach(`div.video-info-title h1 span a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		*out = append(*out, sc)
	})

	siteCollector.OnHTML(`div.wp-pagenavi a.page`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.video-page-block a.model-foto`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://vrbangers.com/videos/")

	siteCollector.Wait()
	sceneCollector.Wait()

	wg.Done()
	return nil
}

func init() {
	registerScraper("vrbangers", "VRBangers", VRBangers)
}
