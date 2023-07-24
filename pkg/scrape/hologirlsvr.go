package scrape

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func HoloGirlsVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	scraperID := "hologirlsvr"
	siteID := "HoloGirlsVR"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.hologirlsvr.com")
	siteCollector := createCollector("www.hologirlsvr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "HoloFilm Productions"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.video-title h3`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Title = e.Text
			}
		})

		// Cover URLs
		e.ForEach(`div.vidCover`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				src := strings.TrimSpace(e.ChildAttr(`img`, "src"))
				sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(src))
			}
		})

		// Gallery
		e.ForEach(`div.vid-flex-container a`, func(id int, e *colly.HTMLElement) {
			href := strings.TrimSpace(e.Attr("href"))
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(href))
		})

		// Synopsis
		r, _ := regexp.Compile("(?s)Synopsis:(.*)Tags:")
		e.ForEach(`div.vidpage-info`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				match := r.FindStringSubmatch(e.Text)
				sc.Synopsis = strings.TrimSpace(match[1])
			}
		})

		// Cast
		e.ForEach(`div.vidpage-featuring span`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.vidpage-mobilePad`, func(id int, e *colly.HTMLElement) {
			ad := models.ActorDetails{Source: sc.ScraperID + " scrape"}
			e.ForEach(`a`, func(id int, e *colly.HTMLElement) {
				ad.ProfileUrl = e.Response.Request.AbsoluteURL(e.Attr("href"))
			})
			e.ForEach(`img.img-responsive`, func(id int, e *colly.HTMLElement) {
				if !strings.HasSuffix(e.Attr("src"), "/missing.jpg") {
					ad.ImageUrl = e.Response.Request.AbsoluteURL(e.Attr("src"))
				}
			})
			e.ForEach(`strong`, func(id int, e *colly.HTMLElement) {
				sc.ActorDetails[strings.TrimSpace(e.Text)] = ad
			})

		})

		// Tags
		e.ForEach(`div.videopage-tags em`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination-container li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div#starIndexData div.featModelCell > a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.modelpage-vidpad a.vidIcons`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://www.hologirlsvr.com/Models")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("hologirlsvr", "HoloGirlsVR", "https://pbs.twimg.com/profile_images/836310876797837312/Wb3-FTxD_200x200.jpg", "hologirlsvr.com", HoloGirlsVR)
}
