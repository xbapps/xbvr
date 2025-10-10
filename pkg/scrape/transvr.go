package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func TransVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "transvr"
	siteID := "TransVR"
	allowedDomains := []string{"transvr.com", "www.transvr.com", "www.groobyod.com"}
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector(allowedDomains...)
	siteCollector := createCollector(allowedDomains...)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "TransVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// https://www.transvr.com/tour/trailers/Peri-Faye-Is-In-My-Bed.html
		// https://www.groobyod.com/ondemand/scenes/Peri-Faye-Is-In-My-Bed_vids.html
		t := strings.NewReplacer("transvr.com/tour/trailers", "groobyod.com/ondemand/scenes", ".html", "_vids.html")
		trailerURL := t.Replace(sc.HomepageURL)

		// Title
		sc.Title = strings.Replace(e.ChildText(`title`), "Trans VR: ", "", -1)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.trailer_toptitle_left a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: scraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Cover URL
		coverURL := e.Request.Ctx.Get("cover")
		sc.Covers = append(sc.Covers, coverURL)

		// Scene ID - get from URL
		tmps := strings.Split(coverURL, "/")
		tmp := strings.Replace(tmps[len(tmps)-1], ".jpg", "", -1)
		sc.SiteID = tmp
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.trailerpage_description p`))

		// Date
		dateString := strings.Replace(e.ChildText(`div.set_meta`), "Added ", "", -1)
		tmpDate, _ := goment.New(dateString, "MMMM D, YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		// Duration
		r := regexp.MustCompile(`(?:(\d{2}):)?(\d{2}):(\d{2})`)
		m := r.FindStringSubmatch(e.ChildText(`div.set_meta`))
		duration := 0
		if len(m) == 4 {
			hours, _ := strconv.Atoi("0" + m[1])
			minutes, _ := strconv.Atoi(m[2])
			duration = hours*60 + minutes
		}
		sc.Duration = duration

		// Gallery
		sc.Gallery = e.ChildAttrs("div.trailerpage_photoblock_fullsize img", "href")

		// Tags
		e.ForEach(`div.set_tags a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// https://www.groobyod.com/ondemand/content//upload/Peri_Faye_is_in_My_Bed/250911/trailer/perifaye.mattyiceee.ns.tvr.PREVIEW.mp4
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: trailerURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality", ContentBaseUrl: "https://www.groobyod.com"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		out <- sc
	})

	siteCollector.OnHTML(`div.videohere`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a", "href"))
		cvr := e.Request.AbsoluteURL(e.ChildAttr("img", "src"))

		if !funk.ContainsString(knownScenes, sceneURL) {
			ctx := colly.NewContext()
			ctx.Put("cover", cvr)
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	siteCollector.OnHTML(`div.pagination li a:not(.active)`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.transvr.com/tour/categories/movies/1/latest/")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("transvr", "TransVR", "https://www.transvr.com/tour/custom_assets/favicon/apple-touch-icon.png", "transvr.com", TransVR)
}
