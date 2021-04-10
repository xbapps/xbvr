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
)

func BadoinkSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com", "kinkvr.com")
	siteCollector := createCollector("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com", "kinkvr.com")
	trailerCollector := cloneCollector(sceneCollector)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "Badoink"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Site ID
		sc.Site = siteID

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = strings.Replace(tmp[len(tmp)-1], "/", "", -1)
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		e.ForEach(`h1.video-title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs for paid videos
		e.ForEach(`div#videoPreviewContainer .video-image-container picture img`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Attr("src"), "?")[0])
			}
		})
		// Cover URLs for free videos
		e.ForEach(`div#videoPreviewContainer dl8-video`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Attr("poster"), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`div#gallery div.gallery-item`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("data-big-image")))
		})

		// Synopsis
		e.ForEach(`p.video-description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`a.video-tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})
		if scraperID == "vrcosplayx" {
			sc.Tags = append(sc.Tags, "Cosplay", "Parody")
		}

		// Cast
		e.ForEach(`a.video-actor-link`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
		})

		// Date
		e.ForEach(`p.video-upload-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(strings.Replace(e.Text, "Uploaded: ", "", -1), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`p.video-duration`, func(id int, e *colly.HTMLElement) {
			content := strings.Replace(strings.Split(e.Attr("content"), "M")[0], "PT", "", -1)
			tmpDuration, err := strconv.Atoi(content)
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		ctx := colly.NewContext()
		ctx.Put("scene", sc)

		trailerCollector.Request("GET", e.Request.URL.String()+"trailer/", nil, ctx, nil)
	})

	trailerCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)

		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				origURL := e.Attr("src")

				// Some scenes had different trailer name "templates". Some videos didn't have trailers and one VRCosplayX (Death Note) was was missing "_" in the name
				fpName3m := strings.Split(strings.Split(strings.Split(strings.Split(origURL, "_trailer")[0], "_3M")[0], "_3m")[0], "3M")[0]
				fpName2m := strings.Split(strings.Split(strings.Split(fpName3m, "_trailer")[0], "_2M")[0], "_2m")[0]
				fpName := strings.Split(strings.Split(strings.Split(fpName2m, "_trailer")[0], "_1M")[0], "_1m")[0]

				fragmentName := strings.Split(fpName, "/")
				baseName := fragmentName[len(fragmentName)-1]

				filenames := []string{"samsung_180_180x180_3dh_LR", "oculus_180_180x180_3dh_LR", "mobile_180_180x180_3dh_LR", "7k_180_180x180_3dh_LR", "5k_180_180x180_3dh_LR", "4k_HEVC_180_180x180_3dh_LR", "ps4_180_sbs", "ps4_pro_180_sbs"}

				for i := range filenames {
					filenames[i] = baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`main[data-page=VideoList] a.video-card-image-container`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.Visit(URL)

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func BadoinkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, "badoinkvr", "BadoinkVR", "https://badoinkvr.com/vrpornvideos")
}

func B18VR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, "18vr", "18VR", "https://18vr.com/vrpornvideos")
}

func VRCosplayX(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, "vrcosplayx", "VRCosplayX", "https://vrcosplayx.com/cosplaypornvideos")
}

func BabeVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, "babevr", "BabeVR", "https://babevr.com/vrpornvideos")
}

func KinkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, "kinkvr", "KinkVR", "https://kinkvr.com/bdsm-vr-videos")
}

func init() {
	registerScraper("badoinkvr", "BadoinkVR", "https://pbs.twimg.com/profile_images/618071358933610497/QaMV81nF_200x200.png", BadoinkVR)
	registerScraper("18vr", "18VR", "https://pbs.twimg.com/profile_images/989481761783545856/w-iKqgqV_200x200.jpg", B18VR)
	registerScraper("vrcosplayx", "VRCosplayX", "https://pbs.twimg.com/profile_images/900675974039298049/ofMytpkQ_200x200.jpg", VRCosplayX)
	registerScraper("babevr", "BabeVR", "https://babevr.com/babevr_icons/apple-touch-icon.png", BabeVR)
	registerScraper("kinkvr", "KinkVR", "https://kinkvr.com/kinkvr_icons/apple-touch-icon.png", KinkVR)
}
