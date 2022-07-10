package scrape

import (
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRPHub(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, company string, vrpCategory string,  callback func(e *colly.HTMLElement, sc *models.ScrapedScene)) error {
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
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})
		if (!isPost) {
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

		// Cast
		e.ForEach(`div.td-post-header header.td-post-title span.td-post-date2 a.ftlink`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, e.Text)
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

		// Filenames
		e.ForEach(`div.td-post-featured-video dl8-video source:not([quality=Default])`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(strings.TrimRight(e.Attr("src"), "/"), "/")
			if len(parts) > 0 {
				sc.Filenames = append(sc.Filenames, parts[len(parts)-1])
			}
		})

		callback(e, sc)
		out <- *sc
	})

	siteCollector.OnHTML(`div.page-nav a.page`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.td-main-content div.td-module-image-main a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		reCover := regexp.MustCompile(`^(.+)-e\d+-\d+x\d+(\.\w+)$`)
		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sc := models.ScrapedScene{}

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

	siteCollector.Visit("https://vrphub.com/category/" + vrpCategory + "/")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

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
	for i := range tmpVideoUrls {
		if sceneIdFound {
			break
		}

		tmp := strings.Split(tmpVideoUrls[i], "/")
		tmp2 := strings.Split(tmp[len(tmp)-1], "_")[0]
		if tmp2 != "VRHush" {
			sc.SiteID = strings.Replace(tmp2, "vrh", "", -1)
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
			sceneIdFound = true
		}
	}
}

func addVRPHubScraper(id string, name string, company string, vrpCategory string, avatarURL string, callback func(e *colly.HTMLElement, sc *models.ScrapedScene)) {
	suffixedName := name + " (VRP Hub)"

	if avatarURL == "" {
		avatarURL = "https://cdn.vrphub.com/wp-content/uploads/2016/08/vrphubnew.png"
	}

	registerScraper(id, suffixedName, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return VRPHub(wg, updateSite, knownScenes, out, id, name, company, vrpCategory, callback)
	})
}

func init() {
	addVRPHubScraper("vrphub-vrhush", "VRHush", "VRHush", "vr-hush", "https://z5w6x5a4.ssl.hwcdn.net/sites/vrh/favicon/apple-touch-icon-180x180.png", vrhushCallback)
}
