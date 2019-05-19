package scrape

import (
	"log"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
)

func ScrapeBadoink(knownScenes []string, out *[]ScrapedScene) error {
	siteCollector := colly.NewCollector(
		colly.AllowedDomains("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	sceneCollector := colly.NewCollector(
		colly.AllowedDomains("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)
	trailerCollector := sceneCollector.Clone()

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	trailerCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Badoink"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Site ID
		if e.Request.URL.Host == "badoinkvr.com" {
			sc.Site = "BadoinkVR"
		}

		if e.Request.URL.Host == "babevr.com" {
			sc.Site = "BabeVR"
		}

		if e.Request.URL.Host == "vrcosplayx.com" {
			sc.Site = "VRCosplayX"
		}

		if e.Request.URL.Host == "18vr.com" {
			sc.Site = "18VR"
		}

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = strings.Replace(tmp[len(tmp)-1], "/", "", -1)
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`h1.video-title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`div#videoPreviewContainer picture source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("srcset")), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`div#gallery div.gallery-item`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("data-big-image")))
		})

		// Synopsis
		e.ForEach(`p.video-description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`a.video-tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Cast
		e.ForEach(`a.video-actor-link`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`p.video-upload-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(strings.Replace(e.Text, "Uploaded: ", "", -1), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`p.video-duration`, func(id int, e *colly.HTMLElement) {
			tmpDuration, err := strconv.Atoi(strings.Split(e.Attr("content"), ":")[1])
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		ctx := colly.NewContext()
		ctx.Put("scene", sc)

		trailerCollector.Request("GET", e.Request.URL.String()+"trailer/", nil, ctx, nil)
	})

	trailerCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(ScrapedScene)

		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				origURL := e.Attr("src")
				fpName := strings.Split(strings.Split(strings.Split(origURL, "_trailer")[0], "_3M")[0], "_3m")[0]
				fragmentName := strings.Split(fpName, "/")
				baseName := fragmentName[len(fragmentName)-1]

				filenames := []string{"samsung_180_180x180_3dh_LR", "oculus_180_180x180_3dh_LR", "mobile_180_180x180_3dh_LR", "5k_180_180x180_3dh_LR"}

				for i := range filenames {
					filenames[i] = baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		*out = append(*out, sc)
	})

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`main[data-page=VideoList] a.video-card-image-container`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://badoinkvr.com/vrpornvideos")
	siteCollector.Visit("https://18vr.com/vrpornvideos")
	siteCollector.Visit("https://vrcosplayx.com/cosplaypornvideos")
	siteCollector.Visit("https://babevr.com/vrpornvideos")

	return nil
}
