package scrape

import (
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/xbapps/xbvr/pkg/models"
)

// Ported from upstream #618 to the current scraper conventions (colly v2,
// ScrapeWG, 5-arg registration). Pornhub won't show scene details without
// login, so scenes are built from the channel listing thumbnails; a
// single-scene URL gets visited best-effort and simply yields nothing.
func PornhubOriginalsVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "pornhuboriginalsvr"
	siteID := "Pornhub Originals VR"
	logScrapeStart(scraperID, siteID)

	siteCollector := createCollector("www.pornhub.com")

	// Pornhub won't let us see scene details without being logged in, so we're using the thumbnail info
	siteCollector.OnHTML(`div.container div.widgetContainer ul.videos li.pcVideoListItem`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sceneId := e.Attr("data-id")
		sceneVKey := e.Attr("data-video-vkey")
		sc.SceneType = "VR"
		sc.Studio = "Pornhub Originals VR"
		sc.Site = siteID
		sc.HomepageURL = "https://www.pornhubpremium.com/view_video.php?viewkey=" + sceneVKey

		sc.SiteID = sceneId
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.thumbnail-info-wrapper a`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Duration
		e.ForEach(`div.marker-overlays var.duration`, func(id int, e *colly.HTMLElement) {
			tmpDurationMins, err := strconv.Atoi(strings.Split(strings.TrimSpace(e.Text), ":")[0])
			if err == nil {
				sc.Duration = tmpDurationMins
			}
		})

		// Cover URLs and parsed release date
		e.ForEach(`div.phimage img.thumb`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Attr("data-thumb_url"))
			tmpDate, _ := goment.New(strings.Join(strings.Split(e.Attr("data-thumb_url"), "/")[4:6], "-"), "YYYYMM-DD")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		out <- sc

	})

	siteCollector.OnHTML(`ul li.page_number a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	if singleSceneURL != "" {
		siteCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.pornhub.com/channels/pornhub-originals-vr/videos?o=da&premium=1")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("pornhuboriginalsvr", "Pornhub Originals VR", "https://di.phncdn.com/pics/sites/000/027/151/avatar1568051926/(m=eidYGe)(mh=bRnuBN2GXIj7F2OV)200x200.jpg", "pornhub.com", PornhubOriginalsVR)
}
