package scrape

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func SexLikeReal(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, company string, siteURL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.sexlikereal.com")
	siteCollector := createCollector("www.sexlikereal.com")

	// RegEx Patterns
	coverRegEx := regexp.MustCompile(`background(?:-image)?\s*?:\s*?url\s*?\(\s*?(.*?)\s*?\)`)
	durationRegExForSceneCard := regexp.MustCompile(`^(?:(\d{2}):)?(\d{2}):(\d{2})$`)
	durationRegExForScenePage := regexp.MustCompile(`^T(\d{0,2})H?(\d{2})M(\d{2})S$`)
	filenameRegEx := regexp.MustCompile(`[:?]|( & )|( \\u0026 )`)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = "slr-" + sc.SiteID

		// Cover
		coverURL := e.ChildAttr(`.splash-screen > img`, "src")
		if len(coverURL) > 0 {
			sc.Covers = append(sc.Covers, coverURL)
		} else {
			m := coverRegEx.FindStringSubmatch(strings.TrimSpace(e.ChildAttr(`.splash-screen`, "style")))
			if len(m) > 0 && len(m[1]) > 0 {
				sc.Covers = append(sc.Covers, m[1])
			}
		}

		// Gallery
		e.ForEach(`meta[name^="twitter:image"]`, func(id int, e *colly.HTMLElement) {
			if e.Attr("name") != "twitter:image" { // we need image1, image2...
				sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("content")))
			}
		})

		// Synopsis
		sc.Synopsis = strings.TrimSpace(
			e.DOM.Find(`div#tabs-about > div > div.u-px--four > div.u-wh`).First().Text())

		// Skipping some very generic and useless tags
		skiptags := map[string]bool{
			"vr porn": true,
			"3D":      true, // Everything gets tagged 3D on SLR, even mono 360
		}

		// Tags
		// Note: known issue with SLR, they use a lot of combined tags like "cheerleader / college / school"
		// ...a lot of these are shared with RealJamVR which uses the same tags though.
		// Could split by / but would run into issues with "f/f/m" and "shorts / skirts"
		var videotype string
		var FB360 string
		e.ForEach(`ul.c-meta--scene-tags li a`, func(id int, e *colly.HTMLElement) {
			if !skiptags[e.Attr("title")] {
				sc.Tags = append(sc.Tags, e.Attr("title"))
			}

			// To determine filenames
			if e.Attr("title") == "Fisheye" || e.Attr("title") == "360°" {
				videotype = e.Attr("title")
			}
			if e.Attr("title") == "Spatial audio" {
				FB360 = "_FB360.MKV"
			}

		})

		// Duration
		sc.Duration = e.Request.Ctx.GetAny("duration").(int)

		// // trailer details
		// sc.TrailerType = "scrape_json"
		// jsonRequest := models.TrailerScrape{SceneUrl: sc.HomepageURL, HtmlElement: "script[type=\"text/javascript\"]", ExtractRegex: "videoData:\\s*(.*}),",
		// 	RecordPath: "src", ContentPath: "url", EncodingPath: "encoding", QualityPath: "quality", ContentBaseUrl: ""}
		// log.Infof("%v", jsonRequest)
		// jsonStr, _ := json.Marshal(jsonRequest)
		// sc.TrailerSrc = string(jsonStr)

		sc.TrailerType = "slr"
		sc.TrailerSrc = "https://api.sexlikereal.com/virtualreality/video/id/" + sc.SiteID

		// Extract from JSON meta data
		// NOTE: SLR only provides certain information like duration as json metadata inside a script element
		// The page code also changes often and is difficult to traverse, best to get as much as possible from metadata
		e.ForEach(`script[type="application/ld+json"]`, func(id int, e *colly.HTMLElement) {
			JsonMetadata := strings.TrimSpace(e.Text)

			// skip non video Metadata
			if gjson.Get(JsonMetadata, "@type").String() == "VideoObject" {

				// Title
				if gjson.Get(JsonMetadata, "name").Exists() {
					sc.Title = strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, "name").String()))
				}

				// Date
				if gjson.Get(JsonMetadata, "datePublished").Exists() {
					sc.Released = gjson.Get(JsonMetadata, "datePublished").String()
				}

				// Cast
				actornames := gjson.Get(JsonMetadata, "actor.#.name")
				for _, name := range actornames.Array() {
					sc.Cast = append(sc.Cast, strings.TrimSpace(html.UnescapeString(name.String())))
				}

				// Duration
				// NOTE: We should already have the duration from the scene list, but if we don't (happens for at least
				// one scene where SLR fails to include it), we try to get it from here
				// We don't always get it from here, because SLR fails to include hours (1h55m30s shows up as T55M30S)
				// ...but this is ready for the format of T01H55M30S should SLR fix that
				if sc.Duration == 0 {
					duration := 0
					if gjson.Get(JsonMetadata, "duration").Exists() {
						tmpParts := durationRegExForScenePage.FindStringSubmatch(gjson.Get(JsonMetadata, "duration").String())
						if len(tmpParts[1]) > 0 {
							if h, err := strconv.Atoi(tmpParts[1]); err == nil {
								hrs := h
								if m, err := strconv.Atoi(tmpParts[2]); err == nil {
									mins := m
									duration = (hrs * 60) + mins
								}
							}
						} else {
							if m, err := strconv.Atoi(tmpParts[2]); err == nil {
								duration = m
							}
						}
						sc.Duration = duration
					}
				}

				// Filenames
				// Only shown for logged in users so need to generate them
				// Format: SLR_siteID_Title_<Resolutions>_SceneID_<LR/TB>_<180/360>.mp4
				resolutions := []string{"_6400p_", "_4000p_", "_3840p_", "_3360p_", "_3160p_", "_3072p_", "_2900p_", "_2880p_", "_2700p_", "_2650p_", "_2160p_", "_1920p_", "_1440p_", "_1080p_", "_original_"}
				baseName := "SLR_" + strings.TrimSuffix(siteID, " (SLR)") + "_" + filenameRegEx.ReplaceAllString(sc.Title, "_")
				switch videotype {
				case "360°": // Sadly can't determine if TB or MONO so have to add both
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MONO_360.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_TB_360.mp4")
					}
				case "Fisheye": // 200° videos named with MKX200
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX200.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX220.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_RF52.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_FISHEYE190.mp4")
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_VRCA220.mp4")
					}
				default: // Assuming everything else is 180 and LR, yet to find a TB_180
					for i := range resolutions {
						sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_LR_180.mp4")
					}
				}
				if FB360 != "" {
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_LR_180"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MKX200"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MKX220"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_RF52"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"FISHEYE190"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_VRCA220"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MONO_360"+FB360)
					sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_TB_360"+FB360)
				}
			}

		})

		out <- sc
	})

	siteCollector.OnHTML(`div.c-pagination ul li a`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`div.c-grid--scenes article`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a[data-qa=scenes-grid-item-link-title]", "href"))
		if strings.Contains(sceneURL, "scene") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				durationText := e.ChildText("div.c-grid-ratio-bottom.u-z--two")
				m := durationRegExForSceneCard.FindStringSubmatch(durationText)
				duration := 0
				if len(m) == 4 {
					hours, _ := strconv.Atoi("0" + m[1])
					minutes, _ := strconv.Atoi(m[2])
					duration = hours*60 + minutes
				}
				ctx := colly.NewContext()
				ctx.Put("duration", duration)
				sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
			}
		}
	})

	siteCollector.Visit(siteURL + "?sort=most_recent")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addSLRScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string) {
	suffixedName := name
	siteNameSuffix := name
	if custom {
		suffixedName += " (Custom SLR)"
		siteNameSuffix += " (SLR)"
	} else {
		if company != "SexLikeReal" {
			suffixedName += " (SLR)"
		}
	}

	if avatarURL == "" {
		avatarURL = "https://www.sexlikereal.com/s/refactor/images/favicons/android-icon-192x192.png"
	}

	registerScraper(id, suffixedName, avatarURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
		return SexLikeReal(wg, updateSite, knownScenes, out, id, siteNameSuffix, company, siteURL)
	})
}

func init() {
	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.SlrScrapers {
		addSLRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL)
	}
	for _, scraper := range scrapers.CustomScrapers.SlrScrapers {
		addSLRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, true, scraper.URL)
	}

}
