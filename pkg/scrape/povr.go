package scrape

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func POVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, company string, siteURL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("povr.com")
	siteCollector := createCollector("povr.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		if scraperID == "" {
			// there maybe no site/studio if user is jusy scraping a scene url
			e.ForEach(`div.meta a[href^="/studios/"]`, func(id int, e *colly.HTMLElement) {
				studioId := strings.TrimSuffix(strings.ReplaceAll(e.Attr("href"), "/studios/", ""), "/")
				sc.Studio = strings.TrimSpace(e.Text)
				sc.Site = sc.Studio
				// see if we can find the site record, there may not be
				db, _ := models.GetDB()
				defer db.Close()
				var site models.Site
				db.Where("name like ?", sc.Studio+"%POVR) or id = ?", sc.Studio, studioId).First(&site)
				if site.ID != "" {
					sc.ScraperID = site.ID
				}
			})
		}
		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = "povr-" + sc.SiteID

		// Title
		e.ForEach(`h1.heading-title`, func(id int, e *colly.HTMLElement) {
			sc.Title = e.Text
		})

		// Date & Duration
		e.ForEach(`div.video__details-grid p.player__date`, func(id int, e *colly.HTMLElement) {
			tmpDetails := strings.Split(e.Text, "  •  ")

			// Date
			tmpDate, _ := goment.New(tmpDetails[1], "DD MMMM, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
			// Duration
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(tmpDetails[0], "min", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// Filenames
		studioBase := path.Base(strings.TrimSuffix(siteURL, "/"))
		sceneBase := path.Base(strings.TrimSuffix(e.Request.URL.Path, "/"))
		sceneBase = strings.Split(sceneBase, sc.SiteID)[0]

		base := studioBase + "-" + sceneBase
		sc.Filenames = append(sc.Filenames, base+"180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"gearvr-180_180x180_3dh_LR.mp4")
		sc.Filenames = append(sc.Filenames, base+"smartphone-180_180x180_3dh_LR.mp4")

		// Cover URLs, and gallery for MilfVR & WankzVR
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
				thumbSizes := map[string]int{"MilfVR": 1280, "WankzVR": 1024}
				if thumbSize, found := thumbSizes[siteID]; found {
					re := regexp.MustCompile(`/[^/]*/[^/]*$`)
					galleryBaseUrl := re.ReplaceAllString(sc.Covers[0], "/thumbs")
					sc.Gallery = make([]string, 6)
					for i := 0; i < 6; i++ {
						sc.Gallery[i] = fmt.Sprintf("%s/%d_%d.jpg", galleryBaseUrl, thumbSize, i+1)
					}
				}
			}
		})

		// Synopsis
		e.ForEach(`div.player__description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`a.btn[href^="/tags/"]`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// trailer details
		sc.TrailerType = "heresphere"
		sc.TrailerSrc = "https://www.povr.com/heresphere/" + sc.SiteID

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a.btn[href^="/pornstars/"]`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: "povr scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.thumbnail-wrap div.thumbnail a.thumbnail__link`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`div.pagination a[class="pagination__page next"]`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(siteURL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addPOVRScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string) {
	suffixedName := name
	siteNameSuffix := name
	if custom {
		suffixedName += " (Custom POVR)"
		siteNameSuffix += " (POVR)"
	} else {
		suffixedName += " (POVR)"
	}
	registerScraper(id, suffixedName, avatarURL, "povr.com", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
		return POVR(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo)
	})
}

func init() {
	registerScraper("povr-single_scene", "POVR - Other Studios", "", "povr.com", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
		return POVR(wg, updateSite, knownScenes, out, singleSceneURL, "", "", "", "", singeScrapeAdditionalInfo)
	})
	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.PovrScrapers {
		addPOVRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL)
	}
	for _, scraper := range scrapers.CustomScrapers.PovrScrapers {
		addPOVRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, true, scraper.URL)
	}
}
