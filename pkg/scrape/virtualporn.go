package scrape

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VirtualPorn(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "bvr"
	siteID := "VirtualPorn"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("virtualporn.com")
	siteCollector := createCollector("virtualporn.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "BangBros"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Title / Cover / ID / Filenames
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Attr("title"))

			tmpCover := e.Request.AbsoluteURL(e.Request.AbsoluteURL(e.Attr("poster")))
			sc.Covers = append(sc.Covers, tmpCover)

			tmp := strings.Split(tmpCover, "/")
			sc.SceneID = strings.Replace(tmp[5], "bvr", "bvr-", 1)

			e.ForEach(`source`, func(id int, e *colly.HTMLElement) {
				tmpFile := strings.Split(e.Attr("src"), "/")
				sc.Filenames = append(sc.Filenames, strings.Replace(tmpFile[len(tmpFile)-1], "trailer-", "", -1))
			})
		})

		file5kExists := false
		for _, filename := range sc.Filenames {
			if strings.Contains(filename, "5k") {
				file5kExists = true
			}
		}
		if !file5kExists {
			sc.Filenames = append(sc.Filenames, strings.Replace(sc.SceneID, "bvr-", "bvr", -1)+"-5k.mp4")
		}

		// Gallery
		e.ForEach(`div.player__thumbs img`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("src"))
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		e.ForEach(`div.player__stats p.player__stats__cast a`, func(id int, e *colly.HTMLElement) {
			if strings.TrimSpace(e.Text) != "" {
				sc.Cast = append(sc.Cast, strings.TrimSpace(strings.ReplaceAll(e.Text, "!", "")))
			}
		})

		// Tags
		e.ForEach(`div.video__tags__list a.tags`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, strings.ToLower(tag))
			}
		})

		// Synposis
		e.ForEach(`p.player__description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Release date / Duration
		tmpDate, _ := goment.New(strings.TrimSpace(e.Request.Ctx.GetAny("date").(string)), "MMM DD, YYYY")
		sc.Released = tmpDate.Format("YYYY-MM-DD")
		tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.Request.Ctx.GetAny("dur").(string), "mins", "", -1)))
		if err == nil {
			sc.Duration = tmpDuration
		}

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination a[class="pagination__link "]`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		if !strings.Contains(pageURL, "s=billing.payment") {
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.recommended__item`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {

			//Date & Duration from main index
			ctx := colly.NewContext()
			e.ForEach(`span.recommended__item__info__date`, func(id int, e *colly.HTMLElement) {
				if id == 0 {
					ctx.Put("date", strings.TrimSpace(e.Text))
				}
			})
			e.ForEach(`span.recommended__item__time`, func(id int, e *colly.HTMLElement) {
				if id == 0 {
					ctx.Put("dur", strings.TrimSpace(e.Text))
				}
			})

			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	siteCollector.Visit("https://virtualporn.com/videos")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("bvr", "VirtualPorn", "https://images.cn77nd.com/members/bangbros/favicon/apple-icon-60x60.png", VirtualPorn)
}
