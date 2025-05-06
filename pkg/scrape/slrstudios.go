package scrape

import (
	"encoding/json"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func absolutegallery(match string) string {
	re := regexp.MustCompile(`(https:\/\/cdn-vr\.(sexlikereal|trannypornvr)\.com\/images\/\d+\/)vr-porn-[\w\-]+?-(\d+)-original(\.webp|\.jpg)`)
	submatches := re.FindStringSubmatch(match)
	if len(submatches) == 0 {
		return match // no match, return original string
	}
	return submatches[1] + submatches[3] + "_o.jpg" // construct new string with desired format
}

func SexLikeReal(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, company string, siteURL string, singeScrapeAdditionalInfo string, limitScraping bool, masterSiteId string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.sexlikereal.com")
	siteCollector := createCollector("www.sexlikereal.com")

	commonDb, _ := models.GetCommonDB()

	// RegEx Patterns
	durationRegExForSceneCard := regexp.MustCompile(`^(?:(\d{2}):)?(\d{2}):(\d{2})$`)
	durationRegExForScenePage := regexp.MustCompile(`^T(\d{0,2})H?(\d{2})M(\d{2})S$`)
	filenameRegEx := regexp.MustCompile(`[:?]|( & )|( \\u0026 )`)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.MasterSiteId = masterSiteId
		if scraperID == "" {
			// there maybe no site/studio if user is just scraping a scene url
			e.ForEach(`div[data-qa="page-scene-studio-name"]`, func(id int, e *colly.HTMLElement) {
				sc.Studio = strings.TrimSpace(e.Text)
				sc.Site = strings.TrimSpace(e.Text)
				studioId := ""
				p := e.DOM.Parent()
				studioId, _ = p.Attr("href")
				studioId = strings.TrimSuffix(strings.ReplaceAll(studioId, "/studios/", ""), "/")

				// see if we can find the site record, there may not be
				var site models.Site
				commonDb.Where("id = ? or name like ? or (name = ? and name like 'SLR%')", studioId, sc.Studio+"%SLR)", sc.Studio).First(&site)
				if site.ID != "" {
					sc.ScraperID = site.ID
				}
			})
		}

		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = "slr-" + sc.SiteID

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
		alphA := "false"
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

			// Passthrough?
			if e.Attr("title") == "Passthrough" || e.Attr("title") == "Passthrough hack" || e.Attr("title") == "Passthrough AR" {
				alphA = "PT"
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
		params := models.TrailerScrape{SceneUrl: "https://api.sexlikereal.com/virtualreality/video/id/" + sc.SiteID}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		s, _ := resty.New().R().
			SetHeader("User-Agent", UserAgent).
			Get(sc.TrailerSrc)
		JsonMetadataA := s.String()

		isTransScene := e.Request.Ctx.GetAny("isTransScene").(bool)

		// Gallery
		e.ForEach(`meta[name^="twitter:image"]`, func(id int, e *colly.HTMLElement) {
			re := regexp.MustCompile(`(https:\/\/cdn-vr\.(sexlikereal|trannypornvr)\.com\/images\/\d+\/)vr-porn-[\w\-]+?-(\d+)-original(\.webp|\.jpg)`)
			if e.Attr("name") != "twitter:image" { // we need image1, image2...
				if !isTransScene {
					sc.Gallery = append(sc.Gallery, re.ReplaceAllStringFunc(e.Request.AbsoluteURL(e.Attr("content")), absolutegallery))
				} //else {
				//	sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("content")))
				//}
				// Trans scenes currently do not scrape gallery images at all
				// I'm not sure how to, since we're using "twitter:image" and there are none for trans scenes
				// The URLs are available on the "Photos" tab and they can be re-written to redirect to original images similarly
				// The RegEx will work for either
				// i.e. "https://cdn-vr.trannypornvr.com/images/861/vr-porn-Welcome-Back-4570.webp" -> "https://cdn-vr.trannypornvr.com/images/861/4570_o.jpg"
			}
		})

		// Cover
		if !isTransScene {
			appCover := gjson.Get(JsonMetadataA, "thumbnailUrl").String()

			if appCover != "" {

				desktopCover := strings.Replace(appCover, "app", "desktop", -1)
				desktopCresp, err := http.Head(desktopCover)

				if err != nil {
					log.Errorln("Method Head Failed on desktopCover", desktopCover, "with error", err)
				} else {
					if desktopCresp.StatusCode == 200 {
						coverURL := desktopCover
						sc.Covers = append(sc.Covers, coverURL)
					} else {
						appCresp, err := http.Head(appCover)
						if err != nil {
							log.Errorln("Method Head Failed on appCover", appCover, "with error", err)
						} else {
							if appCresp.StatusCode == 200 {
								coverURL := appCover
								sc.Covers = append(sc.Covers, coverURL)
							} else {
								e.ForEach(`link[as="image"]`, func(id int, e *colly.HTMLElement) {
									sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("href")))
								})
							}
							defer appCresp.Body.Close()
						}
					}
					defer desktopCresp.Body.Close()
				}
			} else {
				log.Errorln("No thumnailURL available")
			}
		} else {
			posterURLFound := false
			e.ForEach(`script[type="text/javascript"]`, func(id int, e *colly.HTMLElement) {
				if posterURLFound {
					return
				}
				scriptContent := e.Text
				if strings.Contains(scriptContent, "posterURL") {
					startIndex := strings.Index(scriptContent, `"posterURL":"`) + len(`"posterURL":"`)
					endIndex := strings.Index(scriptContent[startIndex:], `"`)
					if startIndex >= 0 && endIndex >= 0 {
						posterURL := scriptContent[startIndex : startIndex+endIndex]
						unescapedURL := strings.ReplaceAll(posterURL, `\`, "")
						sc.Covers = append(sc.Covers, unescapedURL)
						posterURLFound = true
					}
				}
			})
		}

		// straight and trans videos use a different page structure
		if !isTransScene {
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

				}

			})
		} else { // isTransScene
			e.ForEach("script[type=\"text/javascript\"]", func(id int, e *colly.HTMLElement) {
				var re = regexp.MustCompile(`videoData:\s*(.*}),`)
				r := re.FindStringSubmatch(e.Text)
				if len(r) > 0 {
					JsonMetadata := strings.TrimSpace(r[1])

					// Title
					sc.Title = gjson.Get(JsonMetadata, "title").String()

					// Fix Scene ID Collisions
					sc.SceneID = "slr-trans-" + sc.SiteID

					// Duration - Not Available

					// Filenames
					// trans videos don't appear to follow video type naming conventions
					//					appendFilenames(&sc, siteID, filenameRegEx, "", "", alphA)
				}
			})

			// Date
			e.ForEach("time[data-qa=\"page-scene-studio-date\"]", func(id int, e *colly.HTMLElement) {
				sc.Released = e.Attr("datetime")
			})

			// Cast
			e.ForEach("a[data-qa=\"scene-model-list-item-name\"]", func(id int, e *colly.HTMLElement) {
				sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			})
		}

		// Passthrough "chromaKey":{"enabled":false,"hasAlpha":true,"h":0,"opacity":1,"s":0,"threshold":0,"v":0}
		if alphA == "PT" {
			if gjson.Get(JsonMetadataA, "chromaKey").Exists() {
				sc.ChromaKey = gjson.Get(JsonMetadataA, "chromaKey").String()
			}
			alphA = gjson.Get(JsonMetadataA, "chromaKey.hasAlpha").String()
		}

		// Filenames
		appendFilenames(&sc, siteID, filenameRegEx, videotype, FB360, alphA, JsonMetadataA, isTransScene)

		// actor details
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a[data-qa="scene-model-list-item-photo-link-to-profile"]`, func(id int, e_a *colly.HTMLElement) {
			e_a.ForEach(`img[data-qa="scene-model-list-item-photo-img"]`, func(id int, e_img *colly.HTMLElement) {
				name := e_img.Attr("alt")
				if e_img.Attr("data-src") != "" && name != "" {
					sc.ActorDetails[name] = models.ActorDetails{ImageUrl: e_img.Attr("data-src"), Source: "slr scrape", ProfileUrl: e_a.Request.AbsoluteURL(e_a.Attr("href"))}
				}
			})
		})
		if config.Config.Funscripts.ScrapeFunscripts {
			sc.HasScriptDownload = false
			sc.AiScript = false
			sc.HumanScript = false
			e.ForEach(`ul.c-meta--scene-specs a[href='/tags/sex-toy-scripts-vr']`, func(id int, e_a *colly.HTMLElement) {
				sc.HasScriptDownload = true
				sc.HumanScript = true
			})
			e.ForEach(`ul.c-meta--scene-specs a[href='/tags/sex-toy-scripts-ai-vr']`, func(id int, e_a *colly.HTMLElement) {
				sc.HasScriptDownload = true
				sc.AiScript = true
			})
		}

		out <- sc
	})

	siteCollector.OnHTML(`div.c-pagination ul li a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			WaitBeforeVisit("www.sexlikereal.com", siteCollector.Visit, pageURL)
		}
	})

	siteCollector.OnHTML(`div.c-grid--scenes article`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr("a[data-qa=scenes-grid-item-link-title]", "href"))

		isStraightScene := strings.Contains(sceneURL, "/scene")
		isTransScene := strings.Contains(sceneURL, "/trans")

		if isStraightScene || isTransScene {

			if config.Config.Funscripts.ScrapeFunscripts {
				var existingScene models.Scene

				if masterSiteId == "" {
					commonDb.Where(&models.Scene{SceneURL: sceneURL}).First(&existingScene)
				} else {
					// get the scene from the external_reference table
					var extref models.ExternalReference
					extref.FindExternalUrl("alternate scene "+scraperID, sceneURL)
					var externalData models.SceneAlternateSource
					json.Unmarshal([]byte(extref.ExternalData), &externalData)
					existingScene = externalData.Scene
				}

				if existingScene.ID != 0 || masterSiteId != "" {
					fleshlightBadge := false
					aiBadge := false
					multiBadge := false

					human := false
					ai := false
					e.ForEach(`div.c-grid-badge--fleshlight`, func(id int, e_a *colly.HTMLElement) {
						fleshlightBadge = true
					})
					e.ForEach(`div.c-grid-badge--script-ai`, func(id int, e_a *colly.HTMLElement) {
						ai = true
					})
					e.ForEach(`div.c-grid-badge--fleshlight-badge-multi`, func(id int, e_a *colly.HTMLElement) {
						multiBadge = true
					})

					if fleshlightBadge {
						human = true
					}
					if aiBadge {
						ai = true
					}
					if multiBadge && !fleshlightBadge {
						human = true
						// ai = true
					}

					if existingScene.HumanScript != human || existingScene.AiScript != ai {
						var sc models.ScrapedScene
						sc.InternalSceneId = existingScene.ID
						sc.SceneID = existingScene.SceneID
						sc.ScraperID = scraperID
						sc.HasScriptDownload = true
						sc.OnlyUpdateScriptData = true
						sc.HumanScript = human
						sc.AiScript = ai
						sc.MasterSiteId = masterSiteId
						out <- sc
					}
				}
			}

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
				ctx.Put("isTransScene", isTransScene)
				ScraperRateLimiterWait("www.sexlikereal.com")
				err := sceneCollector.Request("GET", sceneURL, nil, ctx, nil)
				ScraperRateLimiterCheckErrors("www.sexlikereal.com", err)
			}
		}
	})

	if singleSceneURL != "" {
		isTransScene := strings.Contains(singleSceneURL, ".com/trans")
		ctx := colly.NewContext()
		ctx.Put("duration", 0)
		ctx.Put("isTransScene", isTransScene)
		sceneCollector.Request("GET", singleSceneURL, nil, ctx, nil)

	} else {
		WaitBeforeVisit("www.sexlikereal.com", siteCollector.Visit, siteURL+"?sort=most_recent")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func appendFilenames(sc *models.ScrapedScene, siteID string, filenameRegEx *regexp.Regexp, videotype string, FB360 string, AlphA string, JsonMetadataA string, isTransScene bool) {
	// Only shown for logged in users so need to generate them
	// Format: SLR_siteID_Title_<Resolutions>_SceneID_<LR/TB>_<180/360>.mp4
	if !isTransScene {
		// Force siteID when scraping individual scenes without a custom site
		if siteID == "" {
			siteID = gjson.Get(JsonMetadataA, "paysite.name").String()
		}
		viewAngle := gjson.Get(JsonMetadataA, "viewAngle").String()
		projSuffix := "_LR_180.mp4"
		if viewAngle == "190" || viewAngle == "200" || viewAngle == "220" {
			screentype := strings.ToUpper(gjson.Get(JsonMetadataA, "screenType").String())
			projSuffix = "_" + screentype
			if AlphA == "true" {
				projSuffix = "_" + screentype + "_alpha"
			}
			if FB360 != "" {
				FB360 = projSuffix + "_FB360.mkv"
			}
			projSuffix = projSuffix + ".mp4"
		} else if viewAngle == "360" {
			monotb := gjson.Get(JsonMetadataA, "stereomode").String()
			if monotb == "mono" {
				projSuffix = "_MONO_360.mp4"
			} else {
				projSuffix = "_TB_360.mp4"
			}
		}
		resolutions := []string{"_original_"}
		encodings := gjson.Get(JsonMetadataA, "encodings.#(name=h265).videoSources.#.resolution")
		for _, name := range encodings.Array() {
			resolutions = append(resolutions, "_"+name.String()+"p_")
		}
		baseName := "SLR_" + strings.TrimSuffix(siteID, " (SLR)") + "_" + filenameRegEx.ReplaceAllString(strings.ReplaceAll(sc.Title, ":", ";"), "_")
		switch videotype {
		case "360°":
			for i := range resolutions {
				sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+projSuffix)
			}
		case "Fisheye": // 200° videos named with MKX200
			for i := range resolutions {
				sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+projSuffix)
			}
		default: // Assuming everything else is 180 and LR, yet to find a TB_180
			for i := range resolutions {
				sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+projSuffix)
			}
		}
		if FB360 != "" {
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+FB360)
		}
	} else {
		resolutions := []string{"_6400p_", "_4096p_", "_4000p_", "_3840p_", "_3360p_", "_3160p_", "_3072p_", "_3000p_", "_2900p_", "_2880p_", "_2700p_", "_2650p_", "_2160p_", "_1920p_", "_1440p_", "_1080p_", "_original_"}
		baseName := "SLR_" + strings.TrimSuffix(siteID, " (SLR)") + "_" + filenameRegEx.ReplaceAllString(strings.ReplaceAll(sc.Title, ":", ";"), "_")
		switch videotype {
		case "360°": // Sadly can't determine if TB or MONO so have to add both
			for i := range resolutions {
				sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MONO_360.mp4")
				sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_TB_360.mp4")
			}
		case "Fisheye": // 200° videos named with MKX200
			for i := range resolutions {
				if AlphA == "true" {
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX200_alpha.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX220_alpha.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_RF52_alpha.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_FISHEYE190_alpha.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_VRCA220_alpha.mp4")
				} else {
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX200.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_MKX220.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_RF52.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_FISHEYE190.mp4")
					sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_VRCA220.mp4")
				}
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
}

func addSLRScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string, masterSiteId string) {
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

	if masterSiteId == "" {
		registerScraper(id, suffixedName, avatarURL, "sexlikereal.com", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return SexLikeReal(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, "")
		})
	} else {
		registerAlternateScraper(id, suffixedName, avatarURL, "sexlikereal.com", masterSiteId, func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return SexLikeReal(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, masterSiteId)
		})
	}
}

func init() {
	var scrapers config.ScraperList
	// scraper for single scenes with no existing scraper for the studio
	registerScraper("slr-single_scene", "SLR - Other Studios", "", "sexlikereal.com", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return SexLikeReal(wg, updateSite, knownScenes, out, singleSceneURL, "", "", "", "", singeScrapeAdditionalInfo, limitScraping, "")
	})

	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.SlrScrapers {
		addSLRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL, scraper.MasterSiteId)
	}
	for _, scraper := range scrapers.CustomScrapers.SlrScrapers {
		addSLRScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, true, scraper.URL, scraper.MasterSiteId)
	}
}
