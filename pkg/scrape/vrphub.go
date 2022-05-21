package scrape

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRPHub(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, company string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrphub.com")
	siteCollector := createCollector("vrphub.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(e.ChildAttr(`link[rel="shortlink"]`, "href"), "=")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`div.td-post-featured-video`, func(id int, e *colly.HTMLElement) {
			// Title
			sc.Title = e.ChildAttr(`dl8-video`, "title")

		// Date
		e.ForEach(`meta[property="article:published_time"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				tmpDate, err := time.Parse(time.RFC3339, e.Attr("content"))
				if err == nil {
					sc.Released = tmpDate.Format("2006-01-02")
				}
			}
		})

		// Duration
		durationRegex := regexp.MustCompile(`([0-9]+):([0-9]+) min`)
		e.ForEach(`div.td-module-meta-info span.td-post-date`, func(id int, e *colly.HTMLElement) {
			m := durationRegex.FindStringSubmatch(e.Text)
			if len(m) == 3 {
				sc.Duration, _ = strconv.Atoi(m[1])
			}
		})

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`a[data-rel=”lightbox”]`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("href"))
		})

		// Synopsis
		e.ForEach(`meta[name="description"]`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Attr("content"))
		})

		// Cast
		e.ForEach(`span.td-post-date2 a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Tags
		e.ForEach(`ul.td-tags li a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			isCast := false
			for _, cast := range sc.Cast {
				if cast == tag {
					isCast = true
					break
				}
			}
			if !isCast {
				sc.Tags = append(sc.Tags, tag)
			}
		})

			// Filenames

			// **** Tonight's Girlfriend ****
			if scraperID == "tonightsgirlfriend" {
				var baseName string

				base := strings.Split(e.ChildAttr(`source`, "src"), "_")
				tngf := strings.HasPrefix(base[1], "tngf")
				if tngf {
					// https://s1.vrphubcloud.com/videos/cut_tngfkarmaalex_smartphonevr60.mp4
					baseName = base[1]
				} else {
					// https://vrphubcloud.com/videos/tgf/tngfannasamteaser_smartphonevr60.mp4
					base1 := strings.Split(base[0], "/")
					baseName = strings.Split(base1[len(base1)-1], "teaser")[0]
				}

				filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4"}
				for i := range filenames {
					sc.Filenames = append(sc.Filenames, baseName+filenames[i])
				}
			}

			// **** VR Hush & VR Allure ****
			if scraperID == "vrphub-vrhush" || scraperID == "vrphub-vrallure" {
				e.ForEach(`div.td-post-featured-video dl8-video source:not([quality=Default])`, func(id int, e *colly.HTMLElement) {
					parts := strings.Split(strings.TrimRight(e.Attr("src"), "/"), "/")
					if len(parts) > 0 {
						sc.Filenames = append(sc.Filenames, parts[len(parts)-1])
					}
				})
			}

			/*
				....FILENAME ROUTINES FOR OTHER STUDIOS HERE....
			*/
		})

		out <- sc
	})

	siteCollector.OnHTML(`head`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.ChildAttr(`link[rel="next"]`, "href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.td-ss-main-content a[rel="bookmark"]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) && !strings.Contains(sceneURL, "/join") {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit("https://vrphub.com/category/" + slugify.Slugify(strings.ReplaceAll(siteID, "'", "")) + "/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addVRPHubScraper(id string, name string, company string, avatarURL string) {
	suffixedName := name
	if company != "vrphub.COM" {
		suffixedName += " (VRPHub)"
	}
	registerScraper(id, suffixedName, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return VRPHub(wg, updateSite, knownScenes, out, id, name, company)
	})
}

func init() {
	addVRPHubScraper("tonightsgirlfriend", "Tonight's Girlfriend VR", "NaughtyAmerica", "https://mcdn.vrporn.com/files/20200404124349/TNGF_LOGO_BLK.jpg")
	addVRPHubScraper("vrphub-vrhush", "VR Hush", "KB Productions", "https://z5w6x5a4.ssl.hwcdn.net/sites/vrh/favicon/apple-touch-icon-180x180.png")
	addVRPHubScraper("vrphub-vrallure", "VR Allure", "KB Productions", "https://z5w6x5a4.ssl.hwcdn.net/sites/vra/favicon/apple-touch-icon-180x180.png")
}
