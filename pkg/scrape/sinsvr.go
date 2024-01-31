package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

func SinsVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "sinsvr"
	siteID := "SinsVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("xsinsvr.com")
	siteCollector := createCollector("xsinsvr.com")

	durationRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?:(?P<h>\d+):)?(?P<m>\d+):(?P<s>\d+)`),           // e.g. 11:11, 1:11:11
		regexp.MustCompile(`(?:(?P<h>\d+) h(?:ou)?rs? )?(?P<m>\d+) m(?:i)?n`), // e.g. 11 mn, 11 min, 1 hr 11 mn, 1 hour 11 min
		regexp.MustCompile(`(?:(?P<h>\d+)')?(?P<m>\d+)"(?P<s>\d+)`),           // e.g. 1"11, 1'11"11
	}

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "SinsVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID / Title
		sc.SiteID = strings.TrimSpace(e.ChildAttrs(`dl8-video`, "data-scene")[0])
		sc.Title = strings.TrimSpace(e.ChildAttrs(`dl8-video`, "title")[0])
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		if len(e.ChildAttrs(`dl8-video`, "poster")) > 0 {
			sc.Covers = append(sc.Covers, e.ChildAttrs(`dl8-video`, "poster")[0])
		}

		// Gallery
		e.ForEach(`div.tn-photo__container div.cell a `, func(id int, e *colly.HTMLElement) {
			tnphotomore := e.ChildAttr(`div`, "class")
			if tnphotomore == "tn-photo" {
				sc.Gallery = append(sc.Gallery, e.Attr(`href`))
			}
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast and Released
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`.video-detail__specs div.cell`, func(id int, e *colly.HTMLElement) {
			c := strings.TrimSpace(e.Text)
			// Cast
			if strings.Contains(c, "Starring") {
				e.ForEach(`.cell a`, func(id int, e *colly.HTMLElement) {
					cast := strings.Split(e.Text, ",")
					sc.Cast = append(sc.Cast, cast...)
					if len(cast) > 1 {
						sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
					}
					sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
				})
			} else {
				// Released - Date Oct 19, 2019
				if strings.Contains(c, "Released") {
					tmpDate, _ := goment.New(strings.TrimSpace(e.ChildText(`.cell span`)), "MMM D, YYYY")
					sc.Released = tmpDate.Format("YYYY-MM-DD")
				}
			}
		})

		// Duration
		durationText := e.ChildText(`div.video-player-container__info div.tn-video-props span`)
		for _, regex := range durationRegexes {
			match := regex.FindStringSubmatchIndex(durationText)
			hours, _ := strconv.Atoi(string(regex.ExpandString([]byte{}, "0$h", durationText, match)))
			minutes, _ := strconv.Atoi(string(regex.ExpandString([]byte{}, "0$m", durationText, match)))
			duration := hours*60 + minutes
			if duration != 0 {
				sc.Duration = duration
				break
			}
		}

		// Tags
		e.ForEach(`.tags-item a`, func(id int, e *colly.HTMLElement) {
			tags := strings.Split(e.Text, " / ")
			if tags[0] == "VR - Virtual-Reality" {
				tags[0] = "180°"
			} else {
				if tags[0] == "Masturbation/Fingering" {
					tags[0] = "Masturbation"
				}
			}
			sc.Tags = append(sc.Tags, tags...)
			if tags[0] == "Cinematographic" {
				sc.Tags = append(sc.Tags, "Voyeur")
			}
		})

		sc.Synopsis = e.ChildText(`div.tabs li.video-detail__desc p`)

		out <- sc
	})

	siteCollector.OnHTML(`nav.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			if !strings.Contains(pageURL, "/join") {
				siteCollector.Visit(pageURL)
			}
		}
	})

	siteCollector.OnHTML(`div.tn-video`, func(e *colly.HTMLElement) {
		studio := e.ChildText("a.author")

		if studio != "By: SinsVR" && studio != "By: Billie Star" && studio != "By: Poisonio" {
			return
		}

		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a.tn-video-name", "href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && strings.Contains(sceneURL, "/video") && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://xsinsvr.com/studio/sinsvr/videos")
		siteCollector.Visit("https://xsinsvr.com/studio/billie-star/videos")
		siteCollector.Visit("https://xsinsvr.com/studio/poisonio/videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("sinsvr", "SinsVR", "https://assets.xsinsvr.com/logo.black.svg", "xsinsvr.com", SinsVR)
}
