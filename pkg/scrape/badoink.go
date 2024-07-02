package scrape

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/config"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
)

func BadoinkSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com", "kinkvr.com")
	siteCollector := createCollector("badoinkvr.com", "babevr.com", "vrcosplayx.com", "18vr.com", "kinkvr.com")
	trailerCollector := cloneCollector(sceneCollector)

	commonDb, _ := models.GetCommonDB()

	var UHD = "NO"

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
		e.ForEach(`div#videoPreviewContainer video`, func(id int, e *colly.HTMLElement) {
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
			if strings.Contains(strings.TrimSpace(e.Text), "7K") {
				UHD = "7K"
			}
			if strings.Contains(strings.TrimSpace(e.Text), "8K") {
				UHD = "8K"
			}

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

		if config.Config.Funscripts.ScrapeFunscripts {
			e.ForEach(`p.video-tags a[href^="/category/funscript"]`, func(id int, e *colly.HTMLElement) {
				sc.HasScriptDownload = true
			})
		}

		ctx := colly.NewContext()
		ctx.Put("scene", sc)

		trailerCollector.Request("GET", e.Request.URL.String()+"trailer/", nil, ctx, nil)
	})

	trailerCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(models.ScrapedScene)

		e.ForEach(`dl8-video source`, func(id int, e *colly.HTMLElement) {
			if id == 0 {

				// This now needs to be made case insensitive (_trailer is now _Trailer)
				origURLtmp := e.Attr("src")
				origURL := strings.ToLower(origURLtmp)

				// Some scenes had different trailer name "templates". Some videos didn't have trailers and one VRCosplayX (Death Note) was was missing "_" in the name

				fpName3m := strings.Split(strings.Split(strings.Split(origURL, "_trailer")[0], "_3m")[0], "3m")[0]
				fpName2m := strings.Split(strings.Split(fpName3m, "_trailer")[0], "_2m")[0]
				fpName := strings.Split(strings.Split(fpName2m, "_trailer")[0], "_1m")[0]

				fragmentName := strings.Split(fpName, "/")
				baseName := fragmentName[len(fragmentName)-1]

				e.ForEach(`a.video-tag`, func(id int, e *colly.HTMLElement) {
					sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
					if strings.Contains(strings.TrimSpace(e.Text), "7K") {
						UHD = "7K"
					}
					if strings.Contains(strings.TrimSpace(e.Text), "8K") {
						UHD = "8K"
					}
				})
				var filenames []string
				switch UHD {
				case "8K":
					filenames = []string{"8k_180_180x180_3dh", "6k_180_180x180_3dh", "5k_180_180x180_3dh", "4k_HEVC_180_180x180_3dh", "8k_180_180x180_3dh_LR", "6k_180_180x180_3dh_LR", "5k_180_180x180_3dh_LR", "4k_HEVC_180_180x180_3dh_LR", "samsung_180_180x180_3dh", "oculus_180_180x180_3dh", "mobile_180_180x180_3dh", "samsung_180_180x180_3dh_LR", "oculus_180_180x180_3dh_LR", "mobile_180_180x180_3dh_LR", "ps4_180_sbs", "ps4_pro_180_sbs"}
				case "7K":
					filenames = []string{"7k_180_180x180_3dh", "6k_180_180x180_3dh", "5k_180_180x180_3dh", "4k_HEVC_180_180x180_3dh", "7k_180_180x180_3dh_LR", "6k_180_180x180_3dh_LR", "5k_180_180x180_3dh_LR", "4k_HEVC_180_180x180_3dh_LR", "samsung_180_180x180_3dh", "oculus_180_180x180_3dh", "mobile_180_180x180_3dh", "samsung_180_180x180_3dh_LR", "oculus_180_180x180_3dh_LR", "mobile_180_180x180_3dh_LR", "ps4_180_sbs", "ps4_pro_180_sbs"}
				default:
					filenames = []string{"samsung_180_180x180_3dh", "oculus_180_180x180_3dh", "mobile_180_180x180_3dh", "5k_180_180x180_3dh", "4k_HEVC_180_180x180_3dh", "samsung_180_180x180_3dh_LR", "oculus_180_180x180_3dh_LR", "mobile_180_180x180_3dh_LR", "5k_180_180x180_3dh_LR", "4k_HEVC_180_180x180_3dh_LR", "ps4_180_sbs", "ps4_pro_180_sbs"}
				}

				for i := range filenames {
					filenames[i] = baseName + "_" + filenames[i] + ".mp4"
				}

				sc.Filenames = filenames
				sc.Filenames = append(sc.Filenames, baseName+".funscript")

				// Get release date from trailer's creation date if it wasn't on the scene page (BabeVR)
				if sc.Released == "" {
					if trailerURL, err := url.Parse(origURLtmp); err == nil {
						trailerPath := filepath.Join(common.CacheDir, filepath.Base(trailerURL.Path))
						// 200kB should be enough to include the relevant metadata
						if r, err := resty.New().R().SetOutput(trailerPath).SetHeader("Range", "bytes=0-200000").Get(trailerURL.String()); err == nil {
							if probeData, err := ffprobe.GetProbeData(trailerPath, time.Second*10); err == nil {
								if creationTime, err := goment.New(probeData.Format.Tags.CreationTime); err == nil {
									sc.Released = creationTime.Format("YYYY-MM-DD")
								}
							}

							// If we still don't have a release date, try headers sent with the trailer
							// We check both the `date` and `last-modified` headers and take the oldest one
							// This isn't super reliable, but it's better than no date at all
							if sc.Released == "" {
								date, _ := goment.New(r.Header().Get("date"), "ddd, DD MMM YYYY")
								modified, _ := goment.New(r.Header().Get("last-modified"), "ddd, DD MMM YYYY")
								if date == nil || (modified != nil && modified.IsBefore(date)) {
									date = modified
								}
								if date != nil {
									sc.Released = date.Format("YYYY-MM-DD")
								}
							}
						}
						os.Remove(trailerPath)
					}
				}

			}
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`main[data-page=VideoList] a.video-card-image-container`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if config.Config.Funscripts.ScrapeFunscripts {
		siteCollector.OnHTML(`div.video-card-info`, func(e *colly.HTMLElement) {
			sceneURL := ""
			e.ForEach(`div.video-card-info-main a`, func(id int, e *colly.HTMLElement) {
				sceneURL = e.Request.AbsoluteURL(e.Attr("href"))
			})

			e.ForEach(`a[href^="/category/funscript"]`, func(id int, e *colly.HTMLElement) {
				var existingScene models.Scene
				commonDb.Where(&models.Scene{SceneURL: sceneURL}).First(&existingScene)
				if existingScene.ID != 0 && existingScene.ScriptPublished.IsZero() {
					var sc models.ScrapedScene
					sc.InternalSceneId = existingScene.ID
					sc.HasScriptDownload = true
					sc.OnlyUpdateScriptData = true
					sc.HumanScript = false
					sc.AiScript = false
					out <- sc
				}
			})
		})
	}

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

func BadoinkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "badoinkvr", "BadoinkVR", "https://badoinkvr.com/vrpornvideos?order=newest", singeScrapeAdditionalInfo, limitScraping)
}

func B18VR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "18vr", "18VR", "https://18vr.com/vrpornvideos?order=newest", singeScrapeAdditionalInfo, limitScraping)
}

func VRCosplayX(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrcosplayx", "VRCosplayX", "https://vrcosplayx.com/cosplaypornvideos?order=newest", singeScrapeAdditionalInfo, limitScraping)
}

func BabeVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "babevr", "BabeVR", "https://babevr.com/vrpornvideos?order=newest", singeScrapeAdditionalInfo, limitScraping)
}

func KinkVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return BadoinkSite(wg, updateSite, knownScenes, out, singleSceneURL, "kinkvr", "KinkVR", "https://kinkvr.com/bdsm-vr-videos?order=newest", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("badoinkvr", "BadoinkVR", "https://pbs.twimg.com/profile_images/618071358933610497/QaMV81nF_200x200.png", "badoinkvr.com", BadoinkVR)
	registerScraper("18vr", "18VR", "https://pbs.twimg.com/profile_images/989481761783545856/w-iKqgqV_200x200.jpg", "18vr.com", B18VR)
	registerScraper("vrcosplayx", "VRCosplayX", "https://pbs.twimg.com/profile_images/900675974039298049/ofMytpkQ_200x200.jpg", "vrcosplayx.com", VRCosplayX)
	registerScraper("babevr", "BabeVR", "https://babevr.com/icons/babevr/apple-touch-icon.png", "babevr.com", BabeVR)
	registerScraper("kinkvr", "KinkVR", "https://kinkvr.com/icons/kinkvr/apple-touch-icon.png", "kinkvr.com", KinkVR)
}
