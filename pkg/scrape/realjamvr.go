package scrape

import (
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
)

func ScrapeRealJamVR(knownScenes []string, out *[]ScrapedScene) error {
	siteCollector := colly.NewCollector(
		colly.AllowedDomains("realjamvr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("realjamvr.com"),
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
		sc.Studio = "Real Jam Network"
		sc.Site = "RealJam VR"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = strings.Split(tmp[len(tmp)-1], "-")[0]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cast
		e.ForEach(`.featuring a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Duration
		sc.Duration, _ = strconv.Atoi(strings.Split(strings.TrimSpace(e.ChildText(`.duration`)), ":")[0])

		// Released
		sc.Released = strings.TrimSuffix(strings.TrimSpace(e.ChildText(`.date`)), ",")

		// Title, Cover URL
		sc.Title = strings.TrimSpace(e.ChildAttr(`deo-video`, "title"))
		sc.Covers = append(sc.Covers, strings.TrimSpace(e.ChildAttr(`deo-video`, "cover-image")))

		// Gallery
		e.ForEach(`.scene-previews-container a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(e.Attr("href")))
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.desc`))

		// Tags
		e.ForEach(`div.tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Filenames
		set := make(map[string]struct{})
		e.ForEach(`.downloads a`, func(id int, e *colly.HTMLElement) {
			u, _ := url.Parse(e.Attr("href"))
			q := u.Query()
			r, _ := regexp.Compile("attachment; filename=\"(.*?)\"")
			match := r.FindStringSubmatch(q.Get("response-content-disposition"))
			if len(match) > 0 {
				set[match[1]] = struct{}{}
			}
		})
		for f, _ := range set {
			sc.Filenames = append(sc.Filenames, strings.ReplaceAll(f, " ", "_"))
		}

		*out = append(*out, sc)
	})

	siteCollector.OnHTML(`#pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.movies-list a:not(.promo__info):not(#pagination a)`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	return siteCollector.Visit("https://realjamvr.com/virtualreality/list")
}
