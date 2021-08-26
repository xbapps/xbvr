package scrape

import (
	"regexp"
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
		sc.SceneType = "VR"
		sc.Studio = "SinsVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID / Title
		sc.SiteID = strings.TrimSpace(e.ChildAttrs(`dl8-video`, "data-scene")[0])
		sc.Title = strings.TrimSpace(e.ChildAttrs(`dl8-video`, "title")[0])
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		//Cover
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

		//Cast and Released
		e.ForEach(`.video-detail__specs div.cell`, func(id int, e *colly.HTMLElement) {
			c := strings.TrimSpace(e.Text)
			// Cast
			if strings.Contains(c, "Starring") {
				e.ForEach(`.cell a`, func(id int, e *colly.HTMLElement) {
					cast := strings.Split(e.Text, ",")
					sc.Cast = append(sc.Cast, cast...)
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

		//Tags
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
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !strings.Contains(pageURL, "/join") {
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.tn-video a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && strings.Contains(sceneURL, "/video") && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://xsinsvr.com/studio/sinsvr/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("sinsvr", "SinsVR", "https://assets.xsinsvr.com/logo.black.svg", SinsVR)
}
