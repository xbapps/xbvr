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

func SinsVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	logScrapeStart("sinsvr", "SinsVR")

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("sinsvr.com", "www.sinsvr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("sinsvr.com", "www.sinsvr.com"),
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
		sc.Studio = "SinsVR"
		sc.Site = "SinsVR"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = strings.Split(tmp[len(tmp)-1], "-")[0]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		sc.Title = e.Request.Ctx.Get("title")
		sc.Covers = append(sc.Covers, e.Request.Ctx.Get("cover"))

		sc.Gallery = e.ChildAttrs(`img[itemprop="thumbnail"]`, "data-srcset")

		e.ForEach(`.c-video-meta span`, func(id int, e *colly.HTMLElement) {
			c := e.Attr("class")
			if strings.Contains(c, "u-mr--nine") {
				// Oct 19, 2019
				tmpDate, _ := goment.New(strings.TrimSpace(e.Text), "MMM D, YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			} else {
				tmpDuration, err := strconv.Atoi(strings.Split(e.Text, " ")[0])
				if err == nil {
					sc.Duration = tmpDuration
				}
			}
		})

		e.ForEach(`a.u-base`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		e.ForEach(`.c-video-tags li`, func(id int, e *colly.HTMLElement) {
			tags := strings.Split(e.Text, " / ")
			sc.Tags = append(sc.Tags, tags...)
		})

		sc.Synopsis = e.ChildText(`p.u-lh--opt`)

		out <- sc
	})

	siteCollector.OnHTML(`.u-none--tablet-wd a[title]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		ctx := colly.NewContext()
		ctx.Put("title", e.Attr("title"))
		ctx.Put("cover", e.ChildAttr(`img`, "data-srcset"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	siteCollector.OnHTML(`.u-none--tablet-wd .c-pagination ul a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !strings.Contains(pageURL, "/join") {
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.Visit("https://sinsvr.com/virtualreality/list/sort/Recent")

	if updateSite {
		updateSiteLastUpdate("sinsvr")
	}
	logScrapeFinished("sinsvr", "SinsVR")
	return nil
}

func init() {
	registerScraper("sinsvr", "SinsVR", SinsVR)
}
