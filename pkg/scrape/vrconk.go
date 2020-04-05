package scrape

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
	"mvdan.cc/xurls/v2"
)

func VRCONK(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vrconk"
	siteID := "VRCONK"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrconk.com")
	siteCollector := createCollector("vrconk.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VRCONK"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		s := strings.Split(tmp[len(tmp)-1], "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SiteID = s[0]

		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		rxRelaxed := xurls.Relaxed()
		sc.Title = strings.TrimSpace(e.ChildText(`div.item-tr-inner-col h1`))
		sc.Covers = append(sc.Covers, rxRelaxed.FindString(e.ChildAttr(`div.splash-screen`, "style")))

		e.ForEach(`.gallery-block figure > a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("href")))
		})

		e.ForEach(`.stats-list li`, func(id int, e *colly.HTMLElement) {
			// <li><span class="icon i-clock"></span><span class="sub-label">40:54</span></li>
			c := e.ChildAttr(`span`, "class")
			if strings.Contains(c, "i-clock") {
				tmpDuration, err := strconv.Atoi(strings.Split(e.ChildText(`.sub-label`), ":")[0])
				if err == nil {
					sc.Duration = tmpDuration
				}
			}

			if strings.Contains(c, "i-calendar") {
				tmpDate, _ := goment.New(e.ChildText(`.sub-label`))
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}
		})

		// Tags and Cast
		unfilteredTags := []string{}
		e.ForEach(`.tags-block`, func(id int, e *colly.HTMLElement) {
			c := e.ChildText(`.sub-label`)
			if strings.Contains(c, "Categories:") || strings.Contains(c, "Tags:") {
				e.ForEach(`a`, func(id int, ce *colly.HTMLElement) {
					unfilteredTags = append(unfilteredTags, strings.TrimSpace(ce.Text))
				})
			}

			if strings.Contains(c, "Models:") {
				e.ForEach(`a`, func(id int, ce *colly.HTMLElement) {
					sc.Cast = append(sc.Cast, strings.TrimSpace(ce.Text))
				})
			}

		})

		sc.Tags = funk.FilterString(unfilteredTags, func(t string) bool {
			return !funk.ContainsString(sc.Cast, t)
		})

		out <- sc
	})

	siteCollector.OnHTML(`a[data-mb="shuffle-thumbs"]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/signup") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`nav.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !strings.Contains(pageURL, "/user/join") {
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.Visit("https://vrconk.com/virtualreality/list")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrconk", "VRCONK", "https://vrconk.com/s/favicon/apple-touch-icon.png", VRCONK)
}
