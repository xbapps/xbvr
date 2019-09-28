package scrape

import (
	"log"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/robertkrimen/otto"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"gopkg.in/resty.v1"
)

func ScrapeRealityLovers(knownScenes []string, out *[]ScrapedScene) error {
	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("realitylovers.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "RealityLovers"
		sc.Site = "RealityLovers"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID
		sc.SiteID = e.Request.Ctx.Get("id")
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		sc.Covers = append(sc.Covers, e.Request.Ctx.Get("cover"))

		// Title
		sc.Title = e.Request.Ctx.Get("title")

		// Release date
		sc.Released = e.Request.Ctx.Get("released")

		// Gallery
		e.ForEach(`img.videoClip__Details--galleryItem`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, strings.Fields(e.Attr("data-big"))[0])
		})

		// Tags
		e.ForEach(`.videoClip__Details__categoryTag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Synopsis
		e.ForEach(`p[itemprop="description"]`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Duration / Release date / Synopsis
		e.ForEach(`script[type='application/ld+json'][class!='yoast-schema-graph']`, func(id int, e *colly.HTMLElement) {
			vm := otto.New()

			script := "sin = " + e.Text
			script = script + ";\nduration = sin['duration']; datePublished = sin['datePublished']; desc = sin['description'];"
			script = script + "cast = []; sin['actors'].map(function(o){cast.push(o.url)});"

			vm.Run(script)

			out1, _ := vm.Get("duration")
			duration, _ := out1.ToString()
			sc.Duration, _ = strconv.Atoi(strings.Split(duration, ":")[0])

			out2, _ := vm.Get("datePublished")
			relDate, _ := out2.ToString()
			sc.Released = relDate

			out3, _ := vm.Get("desc")
			desc, _ := out3.ToString()
			sc.Synopsis = desc

			out4, _ := vm.Get("cast")
			cast, _ := out4.Export()
			castx, ok := cast.([]string)

			if ok {
				for i := range castx {
					tmpCast = append(tmpCast, castx[i])
				}
			}
		})

		*out = append(*out, sc)
	})

	// Request scenes via REST API
	r, err := resty.R().
		SetHeader("User-Agent", userAgent).
		SetHeader("content-type", "application/json;charset=UTF-8").
		SetHeader("accept", "application/json, text/plain, */*").
		SetHeader("referer", "https://realitylovers.com/videos").
		SetHeader("origin", "https://realitylovers.com").
		SetHeader(":authority", "realitylovers.com").
		SetBody(`{"searchQuery":"","categoryId":null,"perspective":null,"actorId":null,"offset":"5000","isInitialLoad":true,"sortBy":"NEWEST","videoView":"MEDIUM","device":"DESKTOP"}`).
		Post("https://realitylovers.com/videos/search")
	if err == nil || r.StatusCode() == 200 {
		result := gjson.Get(r.String(), "contents")
		result.ForEach(func(key, value gjson.Result) bool {
			sceneURL := "https://realitylovers.com/" + value.Get("videoUri").String()
			if !funk.ContainsString(knownScenes, sceneURL) {
				ctx := colly.NewContext()
				ctx.Put("cover", strings.Fields(value.Get("mainImageSrcset").String())[0])
				ctx.Put("id", value.Get("videoUri").String())
				ctx.Put("released", value.Get("released").String())
				ctx.Put("title", value.Get("title").String())
				sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
			}
			return true
		})
	}

	return nil
}
