package scrape

import (
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// Helper function to get video name from URL
func getVideoName(fileUrl string) (string, error) {
	u, err := url.Parse(fileUrl)
	if err != nil {
		return "", err
	}
	filename := path.Base(u.Path)
	if !strings.Contains(filename, ".") {
		return "", errors.New("filename is not valid")
	}
	return filename, nil
}

func VRPHub(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, company string, siteURL string, singeScrapeAdditionalInfo string, limitScraping bool, callback func(e *colly.HTMLElement, sc *models.ScrapedScene)) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrphub.com")
	siteCollector := createCollector("vrphub.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(*models.ScrapedScene)
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		if scraperID == "" {
			// there maybe no site/studio if user is jusy scraping a scene url
			e.ForEach(`li.entry-category a`, func(id int, e *colly.HTMLElement) {
				studioId := strings.TrimSuffix(strings.ReplaceAll(e.Attr("href"), "https://vrphub.com/category/", ""), "/")
				sc.Studio = strings.TrimSpace(e.Text)
				sc.Site = sc.Studio
				// see if we can find the site record, there may not be
				db, _ := models.GetDB()
				defer db.Close()
				var site models.Site
				db.Where("name like ?", sc.Studio+"%VRPHub) or id = ?", sc.Studio, studioId).First(&site)
				if site.ID != "" {
					sc.ScraperID = site.ID
				}
			})
			// if doing a single scrape we need a cover as well
			e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
				sc.Covers = append(sc.Covers, e.Attr("poster"))
			})

		}

		isPost := false
		e.ForEach(`link[rel="shortlink"]`, func(id int, e *colly.HTMLElement) {
			// This is the link that contains the internal post id for VRPHub.
			// If this doesn't exist, it means we're on a list page instead of
			// a post page
			postUrl := e.Attr("href")
			u, err := url.Parse(postUrl)
			if err != nil {
				return
			}
			isPost = true
			sc.SiteID = u.Query()["p"][0]
			sc.SceneID = "vrphub-" + sc.SiteID
		})
		if !isPost {
			return
		}

		// Title
		e.ForEach(`div.td-post-header header.td-post-title h1.entry-title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Date
		e.ForEach(`div.td-post-header header.td-post-title span.td-post-date time.entry-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMMM D, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.td-post-header header.td-post-title span.td-post-date2 a.ftlink`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, e.Text)
			sc.ActorDetails[e.Text] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Attr("href")}
		})

		// Gallery
		e.ForEach(`div.td-main-content a[data-rel=”lightbox”]`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Attr("href"))
		})

		// Synopsis
		e.ForEach(`div.td-main-content h5 p`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.td-main-content div.td-post-source-tags ul.td-tags li a`, func(id int, e *colly.HTMLElement) {
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

		// Duration
		sc.Duration = 0

		// There are 2 places we can find filenames from - one is in the video
		// previews, and one is in the trailer download section. Some posts
		// list filenames in both, and some only list filenames in 1 of them.
		// We will keep a map of video names to deduplicate filenames from
		// both places
		filenames := map[string]bool{}
		e.ForEach(`div.td-post-featured-video dl8-video source`, func(id int, e *colly.HTMLElement) {
			filename, err := getVideoName(e.Attr("src"))
			if err != nil {
				return
			}
			filenames[filename] = true
		})
		e.ForEach(`div.td-ss-main-content a.maxbutton:not(.maxbutton-get-the-full-video-now)`, func(id int, e *colly.HTMLElement) {
			filename, err := getVideoName(e.Attr("href"))
			if err != nil {
				return
			}
			filenames[filename] = true
		})
		// Insert the deduped filenames to scene
		for filename := range filenames {
			sc.Filenames = append(sc.Filenames, filename)
		}

		callback(e, sc)
		out <- *sc
	})

	siteCollector.OnHTML(`div.page-nav a.page`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.td-main-content div.td-module-image-main a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		reCover := regexp.MustCompile(`^(.+)-e\d+-\d+x\d+(\.\w+)$`)
		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sc := models.ScrapedScene{}
			sc.ScraperID = scraperID

			e.ForEach(`img.entry-thumb-main`, func(id int, e *colly.HTMLElement) {
				cover := e.Attr("src")
				tmpParts := reCover.FindStringSubmatch(cover)
				if tmpParts != nil {
					cover = tmpParts[1] + tmpParts[2]
				}
				sc.Covers = append(sc.Covers, cover)
			})

			ctx := colly.NewContext()
			ctx.Put("scene", &sc)

			sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
		}
	})

	if singleSceneURL != "" {
		sc := models.ScrapedScene{}
		ctx := colly.NewContext()
		ctx.Put("scene", &sc)
		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)
	} else {
		siteCollector.Visit(siteURL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

// We can pass this noop callback for studios that require no modifications
func noop(e *colly.HTMLElement, sc *models.ScrapedScene) {}

func vrhushCallback(e *colly.HTMLElement, sc *models.ScrapedScene) {
	// Scene ID - get from videos
	var tmpVideoUrls []string
	e.ForEach(`div.td-post-featured-video dl8-video`, func(id int, e *colly.HTMLElement) {
		tmpVideoUrls = append(tmpVideoUrls, e.Attr("poster"))
		e.ForEach(`source`, func(id int, e *colly.HTMLElement) {
			tmpVideoUrls = append(tmpVideoUrls, e.Attr("src"))
		})
	})

	sceneIdFound := false
	vrhIdRegEx := regexp.MustCompile(`vrh(\d+)_`)
	for i := range tmpVideoUrls {
		if sceneIdFound {
			break
		}

		matches := vrhIdRegEx.FindStringSubmatch(tmpVideoUrls[i])
		if len(matches) > 0 && len(matches[1]) > 0 {
			sc.SiteID = matches[1]
			sc.SceneID = "vrphub-" + sc.SiteID
			sceneIdFound = true
		}
	}
}

func stripzvrCallback(e *colly.HTMLElement, sc *models.ScrapedScene) {
	// Remove prefix for StripzVR trailers
	for i := range sc.Filenames {
		sc.Filenames[i] = strings.TrimPrefix(sc.Filenames[i], "StripzVR-SAMPLE-")
	}
}

func addVRPHubScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string, callback func(e *colly.HTMLElement, sc *models.ScrapedScene)) {
	suffixedName := name + " (VRP Hub)"
	siteNameSuffix := name
	if custom {
		suffixedName = name + " (Custom VRP Hub)"
		siteNameSuffix += " (VRP Hub)"
	}

	if avatarURL == "" {
		avatarURL = "https://cdn.vrphub.com/wp-content/uploads/2016/08/vrphubnew.png"
	}

	registerScraper(id, suffixedName, avatarURL, "vrphub.com", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return VRPHub(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, callback)
	})
}

func init() {
	registerScraper("vrphub-single_scene", "VRPHub - Other Studios", "", "vrphub.com", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return VRPHub(wg, updateSite, knownScenes, out, singleSceneURL, "", "", "", "", singeScrapeAdditionalInfo, limitScraping, noop)
	})
	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.VrphubScrapers {
		switch scraper.ID {
		case "vrphub-vrhush":
			addVRPHubScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL, vrhushCallback)
		case "vrphub-stripzvr":
			addVRPHubScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL, stripzvrCallback)
		}
		addVRPHubScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL, noop)
	}
	for _, scraper := range scrapers.CustomScrapers.VrphubScrapers {
		addVRPHubScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, true, scraper.URL, noop)
	}
}
