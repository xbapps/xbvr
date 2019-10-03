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

func VirtualRealPorn(knownScenes []string, out *[]ScrapedScene) error {
	siteCollector := colly.NewCollector(
		colly.AllowedDomains("virtualrealporn.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("virtualrealporn.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	castCollector := colly.NewCollector(
		colly.AllowedDomains("virtualrealporn.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
		colly.AllowURLRevisit(),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	castCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VirtualRealPorn"
		sc.Site = "VirtualRealPorn"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		var tmpCast []string

		// Scene ID - get from URL
		e.ForEach(`link[rel=shortlink]`, func(id int, e *colly.HTMLElement) {
			sc.SiteID = strings.Split(e.Attr("href"), "?p=")[1]
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(strings.Replace(e.Text, " - VirtualRealPorn.com", "", -1))
		})

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`a.w-gallery-tnail`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(strings.Split(e.Attr("href"), "?")[0]))
		})

		// Tags
		e.ForEach(`a.g-btn span`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
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

		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				origURL := e.Attr("src")
				fragmentName := strings.Split(origURL, "/")

				fpName := strings.Split(fragmentName[len(fragmentName)-1], "?")[0]
				fpName = strings.Replace(fpName, ".webm", "", -1)
				fpName = strings.Replace(fpName, ".mp4", "", -1)
				fpName = strings.Replace(fpName, "_-_Trailer", "", -1)
				fpName = strings.Replace(fpName, "_-_Smartphone", "", -1)

				var outFilenames []string
				postfix := []string{"-_180x180_3dh"}

				for i := range postfix {
					outFilenames = append(outFilenames, fpName+"_"+postfix[i]+".mp4")
					outFilenames = append(outFilenames, strings.Replace(fpName, ".com", "", -1)+"_"+postfix[i]+".mp4")
				}

				sc.Filenames = outFilenames
			}
		})

		ctx := colly.NewContext()
		ctx.Put("scene", &sc)

		for i := range tmpCast {
			castCollector.Request("GET", tmpCast[i], nil, ctx, nil)
		}

		*out = append(*out, sc)
	})

	castCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(*ScrapedScene)

		var name string
		e.ForEach(`h1.model-title`, func(id int, e *colly.HTMLElement) {
			name = strings.Split(e.Text, " (")[0]
		})

		var gender string
		e.ForEach(`div.model-info div.one-half div`, func(id int, e *colly.HTMLElement) {
			if strings.Split(e.Text, " ")[0] == "Gender" {
				gender = strings.Split(e.Text, " ")[1]
			}
		})

		if gender == "Female" {
			sc.Cast = append(sc.Cast, name)
		}
	})

	siteCollector.OnHTML(`a.w-portfolio-item-anchor`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	// Request scenes via ajax interface
	r, err := resty.R().
		SetHeader("User-Agent", userAgent).
		SetHeader("Accept", "application/json, text/javascript, */*; q=0.01").
		SetHeader("Referer", "https://virtualrealporn.com/").
		SetHeader("X-Requested-With", "XMLHttpRequest").
		SetHeader("Authority", "virtualrealporn.com").
		SetFormData(map[string]string{
			"action": "get_videos_list",
			"p":      "1",
			"vpp":    "1000",
			"sq":     "",
			"so":     "date-DESC",
			"pid":    "8",
		}).
		Post("https://virtualrealporn.com/wp-admin/admin-ajax.php")
	if err == nil || r.StatusCode() == 200 {
		urls := gjson.Get(r.String(), "data.movies.#.permalink").Array()
		for i := range urls {
			sceneURL := urls[i].String()
			if !funk.ContainsString(knownScenes, sceneURL) {
				sceneCollector.Visit(sceneURL)
			}
		}
	}

	return siteCollector.Visit("https://virtualrealporn.com/")
}

func init() {
	registerScraper("virtualrealporn", "VirtualRealPorn", VirtualRealPorn)
}
