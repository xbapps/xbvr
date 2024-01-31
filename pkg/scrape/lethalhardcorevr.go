package scrape

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
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

func LethalHardcoreSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("lethalhardcorevr.com", "whorecraftvr.com")
	siteCollector := createCollector("lethalhardcorevr.com", "whorecraftvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "Celestial Productions"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Site ID
		sc.Site = siteID

		// Release Date
		tmpDate, _ := goment.New(e.Request.Ctx.Get("date"), "MM/DD/YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

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

		// trailer details
		sc.TrailerType = "url"
		sc.TrailerSrc = e.ChildAttr(`span.link-player-action-inner a.btn`, `href`)

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
		sc.ActorDetails = make(map[string]models.ActorDetails)
		r := strings.NewReplacer("(", "", ")", "")
		e.ForEach(`div.item-page-details a[data-target="#bodyShotModal"]`, func(id int, e *colly.HTMLElement) {
			img := ""
			e.ForEach(`img`, func(id int, e *colly.HTMLElement) {
				style := e.Attr("style")
				regexPattern := `url\((.*?)\)`
				regex, _ := regexp.Compile(regexPattern)
				matches := regex.FindStringSubmatch(style)
				if len(matches) > 1 {
					img = matches[1]
				} else {
					if e.Attr("src") != "https://imgs1cdn.adultempire.com/res/pm/pixel.gif" {
						img = e.Attr("src")
					}
				}

			})
			e.ForEach(`.overlay small`, func(id int, e *colly.HTMLElement) {
				if id <= 1 {
					sc.Cast = append(sc.Cast, strings.TrimSpace(r.Replace(e.Text)))
					sc.ActorDetails[strings.TrimSpace(r.Replace(e.Text))] = models.ActorDetails{ImageUrl: img}
				}
			})
		})

		// Tags
		e.ForEach(`meta[name=Keywords]`, func(id int, e *colly.HTMLElement) {
			k := strings.Split(e.Attr("content"), ",")
			for i, tag := range k {
				if i >= len(k)-2 {
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

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.scene-list-item`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))

		ctx := colly.NewContext()
		e.ForEach(`p.scene-update-stats a~span`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				ctx.Put("date", strings.TrimSpace(e.Text))
			}
		})

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("date", "")

		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(URL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func LethalHardcoreVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return LethalHardcoreSite(wg, updateSite, knownScenes, out, singleSceneURL, "lethalhardcorevr", "LethalHardcoreVR", "https://lethalhardcorevr.com/lethal-hardcore-vr-scenes.html?studio=95595&sort=released", singeScrapeAdditionalInfo, limitScraping)
}

func WhorecraftVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return LethalHardcoreSite(wg, updateSite, knownScenes, out, singleSceneURL, "whorecraftvr", "WhorecraftVR", "https://lethalhardcorevr.com/lethal-hardcore-vr-scenes.html?studio=95347&sort=released", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("whorecraftvr", "WhorecraftVR", "https://imgs1cdn.adultempire.com/bn/Whorecraft-VR-apple-touch-icon.png", "lethalhardcorevr.com", WhorecraftVR)
	registerScraper("lethalhardcorevr", "LethalHardcoreVR", "https://imgs1cdn.adultempire.com/bn/Lethal-Hardcore-apple-touch-icon.png", "lethalhardcorevr.com", LethalHardcoreVR)
}
