package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func BadoinkSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, singeScrapeAdditionalInfo string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com", "kinkvr.com")
	siteCollector := createCollector("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com", "kinkvr.com")
	trailerCollector := cloneCollector(sceneCollector)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
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

		// trailer details
		sc.TrailerType = "scrape_html"
		params := models.TrailerScrape{SceneUrl: sc.HomepageURL + "trailer", HtmlElement: "dl8-video source", ContentPath: "src", QualityPath: "quality"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a.video-actor-link`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Date
		e.ForEach(`p.video-upload-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(strings.Replace(e.Text, "Uploaded: ", "", -1), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		durationRegex := regexp.MustCompile(`Duration: ([0-9]+) min`)
		e.ForEach(`p.video-duration`, func(id int, e *colly.HTMLElement) {
			m := durationRegex.FindStringSubmatch(e.Text)
			if len(m) == 2 {
				sc.Duration, _ = strconv.Atoi(m[1])
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

				//This now needs to be made case insensitive (_trailer is now _Trailer)
				origURLtmp := e.Attr("src")
				origURL := strings.ToLower(origURLtmp)

				// Some scenes had different trailer name "templates". Some videos didn't have trailers and one VRCosplayX (Death Note) was was missing "_" in the name

				fpName3m := strings.Split(strings.Split(strings.Split(origURL, "_trailer")[0], "_3m")[0], "3m")[0]
				fpName2m := strings.Split(strings.Split(fpName3m, "_trailer")[0], "_2m")[0]
				fpName := strings.Split(strings.Split(fpName2m, "_trailer")[0], "_1m")[0]

				fragmentName := strings.Split(fpName, "/")
				baseName := fragmentName[len(fragmentName)-1]

				filenames := []string{"samsung_180_180x180_3dh", "oculus_180_180x180_3dh", "mobile_180_180x180_3dh", "7k_180_180x180_3dh", "5k_180_180x180_3dh", "4k_HEVC_180_180x180_3dh", "samsung_180_180x180_3dh_LR", "oculus_180_180x180_3dh_LR", "mobile_180_180x180_3dh_LR", "7k_180_180x180_3dh_LR", "5k_180_180x180_3dh_LR", "4k_HEVC_180_180x180_3dh_LR", "ps4_180_sbs", "ps4_pro_180_sbs"}

				for i := range filenames {
					filenames[i] = baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
				sc.Filenames = append(sc.Filenames, baseName+".funscript")
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

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(URL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func BadoinkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "badoinkvr", "BadoinkVR", "https://badoinkvr.com/vrpornvideos", singeScrapeAdditionalInfo)
}

func B18VR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "18vr", "18VR", "https://18vr.com/vrpornvideos", singeScrapeAdditionalInfo)
}

func VRCosplayX(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrcosplayx", "VRCosplayX", "https://vrcosplayx.com/cosplaypornvideos", singeScrapeAdditionalInfo)
}

func BabeVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "babevr", "BabeVR", "https://babevr.com/vrpornvideos", singeScrapeAdditionalInfo)
}

func KinkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "kinkvr", "KinkVR", "https://kinkvr.com/bdsm-vr-videos", singeScrapeAdditionalInfo)
}

func init() {
	registerScraper("badoinkvr", "BadoinkVR", "https://pbs.twimg.com/profile_images/618071358933610497/QaMV81nF_200x200.png", "badoinkvr.com", BadoinkVR)
	registerScraper("18vr", "18VR", "https://pbs.twimg.com/profile_images/989481761783545856/w-iKqgqV_200x200.jpg", "18vr.com", B18VR)
	registerScraper("vrcosplayx", "VRCosplayX", "https://pbs.twimg.com/profile_images/900675974039298049/ofMytpkQ_200x200.jpg", "vrcosplayx.com", VRCosplayX)
	registerScraper("babevr", "BabeVR", "https://babevr.com/babevr_icons/apple-touch-icon.png", "babevr.com", BabeVR)
	registerScraper("kinkvr", "KinkVR", "https://kinkvr.com/kinkvr_icons/apple-touch-icon.png", "kinkvr.com", KinkVR)
}
