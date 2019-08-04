package scrape

import (
	"log"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
)

func ScrapeMilfVR(knownScenes []string, out *[]ScrapedScene) error {
	siteCollector := colly.NewCollector(
		colly.AllowedDomains("www.milfvr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("www.milfvr.com"),
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
		sc := ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Wankz"
		sc.Site = "MilfVR"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.title h2`, func(id int, e *colly.HTMLElement) {
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
		e.ForEach(`div.swiper-slide img`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("src")))
			}
		})

		// Gallery
		e.ForEach(`div.swiper-slide img.swiper-lazy`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("data-src")))
			}
		})

		// Synopsis
		e.ForEach(`p.desc`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`i.icon-tag`, func(id int, e *colly.HTMLElement) {
			e.DOM.Parent().Find(`a`).Each(func(id int, e *goquery.Selection) {
				sc.Tags = append(sc.Tags, e.Text())
			})
		})

		// Cast
		e.ForEach(`i.icon-head`, func(id int, e *colly.HTMLElement) {
			e.DOM.Parent().Find(`a`).Each(func(id int, e *goquery.Selection) {
				sc.Cast = append(sc.Cast, e.Text())
			})
		})

		// Date
		e.ForEach(`i.icon-bell`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.DOM.Parent().Text(), "DD MMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`i.icon-clock`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.DOM.Parent().Text(), "min", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		*out = append(*out, sc)
	})

	siteCollector.OnHTML(`nav.pager a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.contentContainer article a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	return siteCollector.Visit("https://www.milfvr.com/videos")
}
