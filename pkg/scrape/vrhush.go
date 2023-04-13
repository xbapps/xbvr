package scrape

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRHush(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	defer wg.Done()
	scraperID := "vrhush"
	siteID := "VRHush"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrhush.com")
	siteCollector := createCollector("vrhush.com")
	castCollector := createCollector("vrhush.com")
	castCollector.AllowURLRevisit = true

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRHush"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://vrhush.com/scenes/", "https://ma.vrhush.com/scene/", 1)

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		tmp2 := strings.Split(tmp[len(tmp)-1], "_")[0]
		sc.SiteID = strings.Replace(tmp2, "vrh", "", -1)
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Regex for original resolution of gallery
		reGetOriginal := regexp.MustCompile(`^(https?:\/\/b8h6h9v9\.ssl\.hwcdn\.net\/vrh\/)(?:largethumbs|hugethumbs|rollover_large|rollover_huge)(\/.+)-c\d{3,4}x\d{3,4}(\.\w{3,4})$`)

		// Title / Cover
		e.ForEach(`.latest-scene-title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})
		e.ForEach(`web-vr-video-player`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("coverimage")))
		})

		// Gallery
		// note 'rollover_large' could be changed to 'rollover_huge' for HQ original but those are easily 5Mb+
		e.ForEach(`div.owl-carousel img.img-responsive`, func(id int, e *colly.HTMLElement) {
			tmpParts := reGetOriginal.FindStringSubmatch(e.Request.AbsoluteURL(e.Attr("src")))
			if len(tmpParts) > 3 {
				sc.Gallery = append(sc.Gallery, tmpParts[1]+"rollover_large"+tmpParts[2]+tmpParts[3])
			}
		})

		// Synopsis
		e.ForEach(`span.full-description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`p.tag-container a.label-tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// Cast
		var tmpCast []string
		e.ForEach(`h5.latest-scene-subtitle a`, func(id int, e *colly.HTMLElement) {
			tmpCast = append(tmpCast, e.Attr("href"))
		})

		// Date
		e.ForEach(`div.latest-scene-meta-1 div.text-left`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		sc.Duration = 0

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "web-vr-video-player", ContentPath: "src", QualityPath: "quality", ContentBaseUrl: "https:"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Filenames
		e.ForEach(`input.stream-input-box`, func(id int, e *colly.HTMLElement) {
			origURL, _ := url.Parse(e.Attr("value"))
			sc.Filenames = append(sc.Filenames, origURL.Query().Get("name"))
		})

		ctx := colly.NewContext()
		ctx.Put("scene", &sc)

		for i := range tmpCast {
			castCollector.Request("GET", tmpCast[i], nil, ctx, nil)
		}

		out <- sc
	})

	castCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(*models.ScrapedScene)

		var name string
		reDoubleWhitespace := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
		e.ForEach(`h1#model-name`, func(id int, e *colly.HTMLElement) {
			name = strings.TrimSpace(reDoubleWhitespace.ReplaceAllString(e.Text, " "))
		})

		var gender string
		e.ForEach(`ul.model-attributes li`, func(id int, e *colly.HTMLElement) {
			if strings.Split(e.Text, " ")[0] == "Gender" {
				gender = strings.Split(e.Text, " ")[1]
			}
		})

		if gender == "Female" {
			sc.Cast = append(sc.Cast, name)
		}
	})

	siteCollector.OnHTML(`ul.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.row div.col-md-4 p.desc a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://vrhush.com/scenes")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("vrhush", "VRHush", "https://cdn-nexpectation.secure.yourpornpartner.com/sites/vrh/favicon/apple-touch-icon-180x180.png", VRHush)
}
