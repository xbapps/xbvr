package scrape

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func WetVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "wetvr"
	siteID := "WetVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("wetvr.com")
	siteCollector := createCollector("wetvr.com")

	sceneCollector.OnHTML(`div#trailer_player`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "WetVR"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://wetvr.com/", "https://wetvr.com/members/", 1)

		// Scene ID - get from previous page
		sc.SiteID = e.Request.Ctx.GetAny("scene-id").(string)
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`div.scene-info h1`))

		scenedate := e.Request.Ctx.GetAny("scene-date").(string)
		if scenedate != "" {
			tmpDate, _ := goment.New(scenedate, "MM/DD/YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		}

		// Cover URLs
		coverSrc := e.ChildAttr(`div[id="player-wrapper"] deo-video`, "cover-image")
		if coverSrc == "" {
			coverSrc = strings.Split(e.ChildAttr(`div[id="no-player-wrapper"] div.bg-cover`, "style"), "background-image: url(")[1]
			coverSrc = strings.TrimPrefix(coverSrc, "'")
			coverSrc = strings.TrimSuffix(coverSrc, "')")
		}
		if coverSrc != "" {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(coverSrc))
		}

		// Gallery
		e.ForEach(`div.items-center  a[href="/join" ] img`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("src")))
			}
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(e.ChildText(`div.items-start span`))

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "deo-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		e.ForEach(`a[href^="/models/"]`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Tags
		// no tags on this site

		// Filenames
		baseFilename := strings.TrimPrefix(sc.HomepageURL, "https://wetvr.com/video/")
		sc.Filenames = append(sc.Filenames, "wetvr-"+baseFilename+"-2700.mp4")
		sc.Filenames = append(sc.Filenames, "wetvr-"+baseFilename+"-2048.mp4")
		sc.Filenames = append(sc.Filenames, "wetvr-"+baseFilename+"-1600.mp4")
		sc.Filenames = append(sc.Filenames, "wetvr-"+baseFilename+"-960.mp4")

		out <- sc
	})

	siteCollector.OnHTML(`ul a.page-link`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div:has(p:contains("Latest")) div[id^="r-"]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a", "href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {

			// SceneID and release date are only available here on div.card
			ctx := colly.NewContext()
			ctx.Put("scene-id", strings.TrimPrefix(e.Attr("id"), "r-"))
			// get the date if it exists
			pDate := e.DOM.Find(`div.video-thumbnail-footer div>span`)
			if pDate.Length() > 0 {
				ctx.Put("scene-date", pDate.Text())
			} else {
				ctx.Put("scene-date", "")
			}
			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	if singleSceneURL != "" {
		type extraInfo struct {
			FieldName  string `json:"fieldName"`
			FieldValue string `json:"fieldValue"`
		}
		var extrainfo []extraInfo
		json.Unmarshal([]byte(singeScrapeAdditionalInfo), &extrainfo)
		ctx := colly.NewContext()
		ctx.Put("scene-id", extrainfo[0].FieldValue)
		if len(extrainfo) > 1 {
			parsedDate, _ := time.Parse("2006-01-02", extrainfo[1].FieldValue)
			formattedDate := parsedDate.Format("January 02, 2006")
			ctx.Put("scene-date", formattedDate)
		} else {
			ctx.Put("scene-date", "")
		}
		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)
	} else {
		siteCollector.Visit("https://wetvr.com/")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("wetvr", "WetVR", "https://wetvr.com/wetvr-favicone2df70df.ico", "wetvr.com", WetVR)
}
