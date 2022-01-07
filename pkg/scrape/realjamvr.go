package scrape

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func RealJamVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "realjamvr"
	siteID := "RealJam VR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("realjamvr.com")
	siteCollector := createCollector("realjamvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Real Jam Network"
		sc.Site = siteID
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

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`h1`))

		// Cover URL
		re := regexp.MustCompile(`background(?:-image)?\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
		coverURL := re.FindStringSubmatch(strings.TrimSpace(e.ChildAttr(`.splash-screen`, "style")))[1]
		if len(coverURL) > 0 {
			sc.Covers = append(sc.Covers, coverURL)
		}

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
		for f := range set {
			sc.Filenames = append(sc.Filenames, strings.ReplaceAll(strings.ReplaceAll(f, " ", "_"), ":", "_"))
		}

		out <- sc
	})

	siteCollector.OnHTML(`.c-pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.movies-list a:not(.promo__info):not(.c-pagination a)`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://realjamvr.com/virtualreality/list")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("realjamvr", "RealJam VR", "https://styles.redditmedia.com/t5_3iym1/styles/communityIcon_kqzp15xw0r361.png", RealJamVR)
}
