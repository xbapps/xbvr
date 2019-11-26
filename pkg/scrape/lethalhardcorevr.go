package scrape

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func isGoodTag(lookup string) bool {
	switch lookup {
	case
		"vr",
		"whorecraft",
		"video",
		"streaming",
		"porn",
		"movie":
		return false
	}
	return true
}

func LethalHardcoreSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("lethalhardcorevr.com", "whorecraftvr.com")
	siteCollector := createCollector("lethalhardcorevr.com", "whorecraftvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Celestial Productions"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Site ID
		sc.Site = siteID

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-2]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Cover
		e.ForEach(`style`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				html, err := e.DOM.Html()
				if err == nil {
					re := regexp.MustCompile(`background\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
					i := re.FindStringSubmatch(html)[1]
					if len(i) > 0 {
						sc.Covers = append(sc.Covers, re.FindStringSubmatch(html)[1])
					}
				}
			}
		})

		// Title
		e.ForEach(`div.item-page-details h1`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = strings.TrimSpace(e.Text)
			}
		})

		// Gallery
		e.ForEach(`div.screenshots-block img`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, strings.TrimSpace(e.Attr("src")))
		})

		// Synposis
		e.ForEach(`#synopsis-full p`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Synopsis = strings.TrimSpace(e.Text)
			}
		})

		// Cast
		r := strings.NewReplacer("(", "",
			")", "")
		e.ForEach(`div.item-page-details .overlay small`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Cast = append(sc.Cast, strings.TrimSpace(r.Replace(e.Text)))
			}
		})

		// Tags
		e.ForEach(`meta[name=Keywords]`, func(id int, e *colly.HTMLElement) {
			k := strings.Split(e.Attr("content"), ",")
			for i, tag := range k {
				if i == len(k)-1 {
					for _, actor := range sc.Cast {
						if funk.Contains(tag, actor) {
							tag = strings.Replace(tag, actor, "", -1)
						}
					}
				}
				tag = strings.ToLower(strings.TrimSpace(tag))
				if isGoodTag(tag) {
					sc.Tags = append(sc.Tags, tag)
				}
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.poster-grid-item a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit(URL)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func LethalHardcoreVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return LethalHardcoreSite(wg, updateSite, knownScenes, out, "lethalhardcorevr", "LethalHardcoreVR", "https://lethalhardcorevr.com/lethal-hardcore-vr-scenes.html")
}

func WhorecraftVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return LethalHardcoreSite(wg, updateSite, knownScenes, out, "whorecraftvr", "WhorecraftVR", "https://whorecraftvr.com/whorecraft-xxx-vr-3d-campaigns.html")
}

func init() {
	registerScraper("whorecraftvr", "WhorecraftVR", "https://imgs1cdn.adultempire.com/bn/Whorecraft-VR-apple-touch-icon.png", WhorecraftVR)
	registerScraper("lethalhardcorevr", "LethalHardcoreVR", "https://imgs1cdn.adultempire.com/bn/Lethal-Hardcore-apple-touch-icon.png", LethalHardcoreVR)
}
