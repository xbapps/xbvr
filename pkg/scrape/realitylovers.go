package scrape

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func RealityLoversSite(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, domain string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector(domain)
	siteCollector := createCollector(domain)

	// These cookies are needed for age verification.
	siteCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "agreedToDisclaimer=true")
	})

	sceneCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "agreedToDisclaimer=true")
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "RealityLovers"
		sc.Site = siteID
		sc.SiteID = ""
		sc.HomepageURL, _ = strings.CutSuffix(e.Request.URL.String(), "/")

		// Cover Url
		coverURL := e.Request.Ctx.GetAny("coverURL").(string)
		sc.Covers = append(sc.Covers, coverURL)

		// Gallery
		e.ForEach(`div.owl-carousel div.item`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.ChildAttr("img", "src"))
		})

		// Incase we scrape a single scene use one of the gallery images for the cover
		if singleSceneURL != "" {
			sc.Covers = append(sc.Covers, sc.Gallery[0])
		}

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`table.video-description-list tbody`, func(id int, e *colly.HTMLElement) {
			// Cast
			e.ForEach(`tr:nth-child(1) a`, func(id int, e *colly.HTMLElement) {
				if strings.TrimSpace(e.Text) != "" {
					sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
					sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
				}
			})

			// Tags
			e.ForEach(`tr:nth-child(2) a`, func(id int, e *colly.HTMLElement) {
				tag := strings.TrimSpace(e.Text)

				// Standardize the resolution tags
				tag, _ = strings.CutSuffix(strings.ToLower(tag), " vr porn")
				tag, _ = strings.CutSuffix(tag, " ts")
				sc.Tags = append(sc.Tags, tag)
			})

			// Date
			tmpDate, _ := goment.New(strings.TrimSpace(e.ChildText(`tr:nth-child(3) td:last-child`)), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Synposis
		sc.Synopsis = strings.TrimSpace(e.ChildText("div.accordion-body"))

		tmp := strings.Split(sc.HomepageURL, "/")

		// Title
		sc.Title = e.Request.Ctx.GetAny("title").(string)

		//Fall back incase single scene scraping
		if sc.Title == "" {
			sc.Title = strings.ReplaceAll(tmp[len(tmp)-1], "-", " ")
		}

		// Scene ID
		sc.SiteID = tmp[len(tmp)-2]

		if sc.SiteID != "" {
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

			// save only if we got a SceneID
			out <- sc
		}
	})

	siteCollector.OnHTML(`a.page-link[aria-label="Next"]:not(.disabled)`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div#gridView`, func(e *colly.HTMLElement) {

		e.ForEach("div.video-grid-view", func(id int, e *colly.HTMLElement) {

			re := regexp.MustCompile(`.+[jJ][pP][gG]`)
			tmp := strings.Split(e.ChildAttr("img", "srcset"), ",")
			r := re.FindStringSubmatch(tmp[len(tmp)-1])
			coverURL := ""

			if len(r) > 0 {
				coverURL = strings.TrimSpace(r[0])
			} else {
				log.Warnln("Couldn't Find Cover Img in srcset:", tmp)
			}

			title := e.ChildText("p.card-title")

			sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a", "href"))

			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				ctx := colly.NewContext()
				ctx.Put("coverURL", coverURL)
				ctx.Put("title", title)
				sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
			}
		})
	})

	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("coverURL", "")
		ctx.Put("title", "")
		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)
	} else {
		siteCollector.Visit("https://" + domain + "/videos/page1")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func RealityLovers(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return RealityLoversSite(wg, updateSite, knownScenes, out, singleSceneURL, "realitylovers", "RealityLovers", "realitylovers.com", singeScrapeAdditionalInfo, limitScraping)
}

func TSVirtualLovers(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return RealityLoversSite(wg, updateSite, knownScenes, out, singleSceneURL, "tsvirtuallovers", "TSVirtualLovers", "tsvirtuallovers.com", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("realitylovers", "RealityLovers", "http://static.rlcontent.com/shared/VR/common/favicons/apple-icon-180x180.png", "realitylovers.com", RealityLovers)
	registerScraper("tsvirtuallovers", "TSVirtualLovers", "http://static.rlcontent.com/shared/TS/common/favicons/apple-icon-180x180.png", "tsvirtuallovers.com", TSVirtualLovers)
}
