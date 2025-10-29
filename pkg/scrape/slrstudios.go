package scrape

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func absolutegallery(match string) string {
	re := regexp.MustCompile(`(https:\/\/cdn-vr\.sexlikereal\.com\/images\/\d+\/)vr-porn-[\w\-]+?-(\d+)-original(\.webp|\.jpg)`)
	submatches := re.FindStringSubmatch(match)
	if len(submatches) == 0 {
		return match // no match, return original string
	}
	return submatches[1] + submatches[3] + "_o.jpg" // construct new string with desired format
}

// normalizeSLRSlug takes a title/slug segment and removes nonstandard characters,
// leaving only lowercase ascii letters, digits, and single dashes.
func normalizeSLRSlug(s string) string {
	s = strings.ToLower(s)
	// Replace spaces with dashes first
	s = strings.ReplaceAll(s, " ", "-")
	// Keep only [a-z0-9-]; drop other characters (e.g., accented letters, punctuation)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			// drop nonstandard characters
		}
	}
	s = b.String()
	// Collapse multiple dashes
	dash := regexp.MustCompile(`-+`)
	s = dash.ReplaceAllString(s, "-")
	// Trim leading/trailing dashes
	s = strings.Trim(s, "-")
	return s
}

// normalizeSLRSceneURL reconstructs a clean SLR scene URL by decoding percent encodings,
// sanitizing the slug, and preserving the numeric ID.
func normalizeSLRSceneURL(u string) string {
	// Best-effort: ensure we operate on expected path
	const base = "https://www.sexlikereal.com/scenes/"
	// Extract ID: last segment after last '-'
	parts := strings.Split(u, "-")
	if len(parts) == 0 {
		return u
	}
	id := parts[len(parts)-1]
	// Remove any trailing query/fragment from id
	if idx := strings.IndexAny(id, "?#"); idx != -1 {
		id = id[:idx]
	}
	// Extract slug portion between '/scenes/' and '-<id>'
	slug := ""
	if i := strings.Index(u, "/scenes/"); i != -1 {
		after := u[i+len("/scenes/"):]
		if j := strings.LastIndex(after, "-"+id); j != -1 {
			slug = after[:j]
		} else {
			// Fallback: remove trailing id by trimming after last '-'
			if k := strings.LastIndex(after, "-"); k != -1 {
				slug = after[:k]
			} else {
				slug = after
			}
		}
	}
	// URL-decode percent encodings in slug
	if unescaped, err := url.PathUnescape(slug); err == nil {
		slug = unescaped
	}
	clean := normalizeSLRSlug(slug)
	if clean == "" || id == "" {
		return u
	}
	return base + clean + "-" + id
}

func SexLikeReal(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, company string, siteURL string, singeScrapeAdditionalInfo string, limitScraping bool, masterSiteId string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	commonDb, _ := models.GetCommonDB()
	// Concurrency control
	apiWG := sync.WaitGroup{}
	sem := make(chan struct{}, 8) // hard-coded concurrency limit

	// API-based scene processing function
	processSceneFromAPI := func(sceneID string, sceneLabel string, isTransScene bool, duration int) {
		// Use v3 API endpoint with scene label for better data including gallery images
		apiURL := "https://api.sexlikereal.com/v3/scenes/" + sceneLabel

		// Fetch scene data from API
		req := resty.New().R().
			SetHeader("User-Agent", UserAgent).
			SetHeader("Client-Type", "web")

		reqconfig := GetCoreDomain(apiURL) + "-scraper"
		log.Debugf("Using Header/Cookies from %s", reqconfig)
		SetupHtmlRequest(reqconfig, req.RawRequest)

		resp, err := req.
			Get(apiURL)

		if err != nil {
			log.Errorln("Failed to fetch API data for scene", sceneID, ":", err)
			return
		}

		if resp.StatusCode() != 200 {
			log.Errorln("API returned non-200 status for scene", sceneID, ":", resp.StatusCode())
			return
		}

		apiData := resp.String()

		// v3 API wraps scene data in a "data" object
		sceneData := gjson.Get(apiData, "data")
		if !sceneData.Exists() {
			log.Errorln("No data field in API response for scene", sceneID)
			return
		}

		// Parse scene data from API response
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = company
		sc.Site = siteID
		sc.MasterSiteId = masterSiteId
		sc.SiteID = sceneID
		sc.SceneID = "slr-" + sceneID

		if isTransScene {
			sc.SceneID = "slr-trans-" + sceneID
		}

		// Get studio info from API if available
		if sceneData.Get("studio.name").Exists() {
			studioName := sceneData.Get("studio.name").String()
			if studioName != "" && scraperID == "" {
				sc.Studio = studioName
				sc.Site = studioName
			}
		}

		// Basic scene information from API
		sc.Title = strings.TrimSpace(html.UnescapeString(sceneData.Get("title").String()))
		sc.Synopsis = strings.TrimSpace(sceneData.Get("description").String())

		// Log that we're scraping this scene
		log.Infoln("Scraping " + "[" + sc.SceneID + "] " + sc.Title)

		// Date - convert timestamp to date string
		if sceneData.Get("date").Exists() {
			timestamp := sceneData.Get("date").Int()
			if timestamp > 0 {
				sc.Released = time.Unix(timestamp, 0).Format("2006-01-02")
			}
		}

		// Duration from API or fallback to passed duration (convert seconds to minutes)
		if sceneData.Get("fullVideoLength").Exists() {
			secs := sceneData.Get("fullVideoLength").Int()
			sc.Duration = int(secs) / 60
		} else {
			sc.Duration = duration / 60
		}

		// Cast from actors array
		actors := sceneData.Get("actors")
		if actors.Exists() {
			sc.ActorDetails = make(map[string]models.ActorDetails)
			actors.ForEach(func(key, value gjson.Result) bool {
				actorName := strings.TrimSpace(value.Get("name").String())
				if actorName != "" {
					sc.Cast = append(sc.Cast, actorName)

					// Actor details
					thumbnailURL := value.Get("thumbnailUrl").String()
					if thumbnailURL != "" {
						sc.ActorDetails[actorName] = models.ActorDetails{
							ImageUrl: thumbnailURL,
							Source:   "slr api",
						}
					}
				}
				return true
			})
		}

		// Tags from categories
		skiptags := map[string]bool{
			"vr porn": true,
			"3D":      true,
		}

		var videotype string
		var FB360 string
		alphA := "false"

		categories := sceneData.Get("categories")
		if categories.Exists() {
			categories.ForEach(func(key, value gjson.Result) bool {
				tagName := strings.TrimSpace(value.Get("name").String())
				if tagName != "" && !skiptags[strings.ToLower(tagName)] {
					sc.Tags = append(sc.Tags, tagName)

					// Check for special tags that affect filename generation
					if tagName == "Fisheye" || tagName == "360°" {
						videotype = tagName
					}
					if tagName == "Spatial audio" {
						FB360 = "_FB360.MKV"
					}
					if tagName == "Passthrough" || tagName == "Passthrough hack" || tagName == "Passthrough AR" || tagName == "Passthrough AI" {
						alphA = "PT"
					}
				}
				return true
			})
		}

		// Process timestamps from API and save as JSON array
		timeStamps := sceneData.Get("timestamps")
		if timeStamps.Exists() && timeStamps.IsArray() {
			var timestampMap []map[string]interface{}
			timeStamps.ForEach(func(key, value gjson.Result) bool {
				ts := value.Get("timestamp").Int()
				name := strings.TrimSpace(value.Get("name").String())
				if name != "" {
					timestampEntry := map[string]interface{}{
						name: ts,
					}
					timestampMap = append(timestampMap, timestampEntry)
				}
				return true
			})

			if len(timestampMap) > 0 {
				timestampJSON, err := json.Marshal(timestampMap)
				if err == nil {
					sc.Timestamps = string(timestampJSON)
				} else {
					log.Errorln("Failed to marshal timestamps for scene", sceneID, ":", err)
				}
			}
		}

		// Cover image
		thumbnailURL := sceneData.Get("thumbnailUrl").String()
		if thumbnailURL != "" {
			// Try desktop version first
			desktopCover := strings.Replace(thumbnailURL, "app", "desktop", -1)
			desktopResp, err := http.Head(desktopCover)
			if err == nil && desktopResp.StatusCode == 200 {
				sc.Covers = append(sc.Covers, desktopCover)
				desktopResp.Body.Close()
			} else {
				// Fallback to app version
				appResp, err := http.Head(thumbnailURL)
				if err == nil && appResp.StatusCode == 200 {
					sc.Covers = append(sc.Covers, thumbnailURL)
					appResp.Body.Close()
				}
			}
		}

		// Homepage URL construction with sanitation
		if sc.Title != "" {
			cleanTitle := normalizeSLRSlug(sc.Title)
			sc.HomepageURL = "https://www.sexlikereal.com/scenes/" + cleanTitle + "-" + sceneID
		} else {
			sc.HomepageURL = "https://www.sexlikereal.com/scenes/scene-" + sceneID
		}

		// Gallery images from API
		images := sceneData.Get("images")
		if images.Exists() && images.IsArray() {
			images.ForEach(func(key, value gjson.Result) bool {
				imageURL := value.Get("imageUrl").String()
				// Only extract images with "original" in the URL
				if imageURL != "" && strings.Contains(imageURL, "original") {
					sc.Gallery = append(sc.Gallery, imageURL)
				}
				return true
			})
		}

		// Trailer setup
		sc.TrailerType = "slr"
		params := models.TrailerScrape{SceneUrl: apiURL}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Passthrough/ChromaKey data
		if alphA == "PT" {
			if sceneData.Get("passthrough").Exists() {
				chromaKeyData := sceneData.Get("passthrough.chromaKey")
				if chromaKeyData.Exists() {
					// Convert to JSON string for storage
					chromaKeyJSON, _ := json.Marshal(chromaKeyData.Value())
					sc.ChromaKey = string(chromaKeyJSON)
				}
				// Check for alpha channel
				if sceneData.Get("passthrough.alpha.enabled").Bool() {
					alphA = "true"
				} else if sceneData.Get("passthrough.aiAlpha.enabled").Bool() {
					alphA = "true"
				}
			}
		}

		// Filenames - need to convert sceneData back to JSON string for legacy function
		sceneDataJSON := sceneData.String()
		appendFilenames(&sc, siteID, videotype, FB360, alphA, sceneDataJSON, isTransScene)

		// Funscript data
		if config.Config.Funscripts.ScrapeFunscripts {
			sc.HasScriptDownload = false
			sc.AiScript = false
			sc.HumanScript = false

			// Check for funscript availability
			scripts := sceneData.Get("scripts")
			if scripts.Exists() && scripts.IsArray() {
				scripts.ForEach(func(key, value gjson.Result) bool {
					sc.HasScriptDownload = true
					isAi := value.Get("scriptAI").Bool()
					if isAi {
						sc.AiScript = true
					} else {
						sc.HumanScript = true
					}
					return true
				})
			}
		}

		out <- sc
	}

	// Function to extract studio code from URL
	// Returns studio code as a string
	getStudioCode := func(studioURL string) string {
		// Extract the last part of the URL (the studio slug)
		parts := strings.Split(studioURL, "/")
		if len(parts) == 0 {
			return ""
		}

		lastPart := parts[len(parts)-1]
		// Remove query parameters if any
		if idx := strings.Index(lastPart, "?"); idx != -1 {
			lastPart = lastPart[:idx]
		}

		// Check if the URL already contains studio code in new format e.g., https://www.sexlikereal.com/studios/ddfnetworkvr-34
		dashIdx := strings.LastIndex(lastPart, "-")
		if dashIdx != -1 && dashIdx < len(lastPart)-1 {
			potentialCode := lastPart[dashIdx+1:]
			// Verify it's all digits
			if _, err := strconv.Atoi(potentialCode); err == nil {
				// Valid studio code found in URL
				return potentialCode
			}
		}

		// If no studio code in URL, look up the studio name in map e.g., "ddfnetworkvr" -> 34
		studioName := strings.ToLower(lastPart)
		if code, found := studioNameToCode[studioName]; found {
			return strconv.Itoa(code)
		}

		log.Errorln("Studio code not found for:", studioName)
		return ""
	}

	fetchScenesFromAPI := func(studioCode string) {
		if studioCode == "" {
			log.Errorln("Studio code is empty, cannot fetch scenes")
			return
		}

		page := 1
		perPage := 36
		hasMore := true

		for hasMore && (!limitScraping || page == 1) {
			apiURL := "https://api.sexlikereal.com/v3/scenes?studios=" + studioCode + "&perPage=" + strconv.Itoa(perPage) + "&sort=mostRecent&page=" + strconv.Itoa(page)

			req := resty.New().R().
				SetHeader("User-Agent", UserAgent).
				SetHeader("Client-Type", "web")

			reqconfig := GetCoreDomain(apiURL) + "-scraper"
			log.Debugf("Using Header/Cookies from %s", reqconfig)
			SetupHtmlRequest(reqconfig, req.RawRequest)

			resp, err := req.
				Get(apiURL)

			if err != nil {
				log.Errorln("Failed to fetch API scenes for studio", studioCode, "page", page, ":", err)
				break
			}

			if resp.StatusCode() != 200 {
				log.Errorln("API returned non-200 status for studio", studioCode, ":", resp.StatusCode())
				break
			}

			apiData := resp.String()
			scenes := gjson.Get(apiData, "data")

			if !scenes.Exists() || !scenes.IsArray() {
				break
			}

			sceneCount := 0
			scenes.ForEach(func(key, scene gjson.Result) bool {
				sceneCount++
				sceneID := scene.Get("id").String()
				sceneLabel := scene.Get("label").String()
				duration := int(scene.Get("fullVideoLength").Int())

				// Determine if trans scene based on categories
				isTransScene := false
				categories := scene.Get("categories")
				if categories.Exists() && categories.IsArray() {
					categories.ForEach(func(k, cat gjson.Result) bool {
						tagName := strings.ToLower(cat.Get("tag.name").String())
						if tagName == "trans" || tagName == "shemale" || tagName == "transgender" {
							isTransScene = true
							return false
						}
						return true
					})
				}

				// Build scene URL for knownScenes check
				title := scene.Get("title").String()
				cleanTitle := normalizeSLRSlug(title)
				sceneURL := "https://www.sexlikereal.com/scenes/" + cleanTitle + "-" + sceneID

				// Handle funscript updates for existing scenes
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
						human := false
						ai := false

						// Check funscript data from API
						fleshlight := scene.Get("fleshlight")
						if fleshlight.Exists() && fleshlight.IsArray() {
							fleshlight.ForEach(func(k, fs gjson.Result) bool {
								isAi := fs.Get("isAiScript").Bool()
								if isAi {
									ai = true
								} else {
									human = true
								}
								return true
							})
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

				// Process new scenes
				if !funk.ContainsString(knownScenes, sceneURL) && sceneID != "" && sceneLabel != "" {
					sem <- struct{}{}
					apiWG.Add(1)
					go func(id string, label string, trans bool, dur int) {
						defer func() { <-sem; apiWG.Done() }()
						processSceneFromAPI(id, label, trans, dur)
					}(sceneID, sceneLabel, isTransScene, duration)
				}

				return true
			})

			// Check if there are more pages
			if sceneCount < perPage {
				hasMore = false
			} else {
				page++
			}
		}
	}

	if singleSceneURL != "" {
		singleSceneURL = normalizeSLRSceneURL(singleSceneURL)
		isTransScene := strings.Contains(singleSceneURL, ".com/trans")

		// Extract scene ID and label from single scene URL for API processing
		sceneID := ""
		sceneLabel := ""

		// Extract the path after /scenes/
		if idx := strings.Index(singleSceneURL, "/scenes/"); idx != -1 {
			scenePath := singleSceneURL[idx+len("/scenes/"):]
			// Remove any query parameters
			if qIdx := strings.Index(scenePath, "?"); qIdx != -1 {
				scenePath = scenePath[:qIdx]
			}
			// The label is the entire path (e.g., "unwiring-her-desires-68823")
			sceneLabel = scenePath

			// Extract ID from the end
			urlParts := strings.Split(scenePath, "-")
			if len(urlParts) > 0 {
				sceneID = urlParts[len(urlParts)-1]
			}
		}

		if sceneID != "" && sceneLabel != "" {
			sem <- struct{}{}
			apiWG.Add(1)
			go func(id string, label string, trans bool) {
				defer func() { <-sem; apiWG.Done() }()
				processSceneFromAPI(id, label, trans, 0)
			}(sceneID, sceneLabel, isTransScene)
		}

	} else {
		// Get studio code from URL (either from new format or map lookup) and fetch scenes via API
		studioCode := getStudioCode(siteURL)
		if studioCode != "" {
			fetchScenesFromAPI(studioCode)
		} else {
			log.Errorln("Failed to get studio code from URL:", siteURL)
		}
	}

	// Wait for all API processing to complete before finishing
	apiWG.Wait()

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func appendFilenames(sc *models.ScrapedScene, siteID string, videotype string, FB360 string, AlphA string, JsonMetadataA string, isTransScene bool) {
	// Only shown for logged in users so need to generate them
	// Format: SLR_siteID_Title_<Resolutions>_SceneID_<LR/TB>_<180/360>.mp4

	// Filename sanitation map
	filenameReplacements := map[string]string{
		":": ";",
		"/": "⁄",
		"|": "¦",
		"*": "#",
		"?": "¿",
	}

	titleSanitized := sc.Title
	for old, new := range filenameReplacements {
		titleSanitized = strings.ReplaceAll(titleSanitized, old, new)
	}

	if !isTransScene {
		// Force siteID when scraping individual scenes without a custom site
		if siteID == "" {
			siteID = gjson.Get(JsonMetadataA, "studio.name").String()
		}
		resolutions := []string{"_original_"}
		// v3 API uses trailerEncodings instead of encodings
		encodings := gjson.Get(JsonMetadataA, "trailerEncodings.#(name=h265).videoSources.#.resolution")
		for _, name := range encodings.Array() {
			resolution := name.String()
			if resolution == "480" || resolution == "720" {
				continue
			}
			resolutions = append(resolutions, "_"+resolution+"p_")
		}
		baseName := "SLR_" + strings.TrimSuffix(siteID, " (SLR)") + "_" + titleSanitized

		projSuffix := "_LR.mp4"
		viewAngle := gjson.Get(JsonMetadataA, "viewAngle").String()

		if viewAngle == "360" {
			monotb := gjson.Get(JsonMetadataA, "stereomode").String()
			if monotb == "mono" {
				projSuffix = "_MONO_360.mp4"
			} else {
				projSuffix = "_TB_360.mp4"
			}
		} else if videotype == "Fisheye" || viewAngle == "190" {
			projSuffix = "_FISHEYE190.mp4"
		}

		for i := range resolutions {
			sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+projSuffix)
		}

		if FB360 != "" {
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+FB360)
		}
	} else {
		resolutions := []string{"_6400p_", "_4096p_", "_4000p_", "_3840p_", "_3360p_", "_3160p_", "_3072p_", "_3000p_", "_2900p_", "_2880p_", "_2700p_", "_2650p_", "_2160p_", "_1920p_", "_1440p_", "_1080p_", "_original_"}
		baseName := "SLR_" + strings.TrimSuffix(siteID, " (SLR)") + "_" + titleSanitized
		switch videotype {
		case "360°": // Can't determine if TB or MONO so have to add both
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
				sc.Filenames = append(sc.Filenames, baseName+resolutions[i]+sc.SiteID+"_LR.mp4")
			}
		}
		if FB360 != "" {
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_LR_180"+FB360)
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MKX200"+FB360)
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_MKX220"+FB360)
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_RF52"+FB360)
			sc.Filenames = append(sc.Filenames, baseName+"_original_"+sc.SiteID+"_FISHEYE190"+FB360)
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

// studioName:code map for backward compat with old studio URLs
var studioNameToCode = map[string]int{
	"100vr": 156, "33": 343, "3d-pickup": 431, "3dvr": 205, "4k-fantasy": 533, "4kvr": 204, "9-block-productions": 883, "ac-vr": 552, "ad4x": 230, "adptube": 717, "adpvr": 598, "adultprime": 931, "adulttime": 608, "ainovdo": 146, "alec-hardy": 436, "alec-hardy-2d": 730,
	"alex-romero": 869, "alexnashvr": 411, "all-anal-vr": 476, "all-group-sex": 770, "allwith": 463, "amanda-thickk": 427, "amateurcouplesvr": 272, "amateurvr3d": 73, "amg-content": 818, "amyvr": 628, "anal-delight": 253, "anal-vault": 760, "analized": 432,
	"angelo-godshack-original": 750, "animeshinclub": 576, "anissa-kate-unleashed": 861, "april-movie-productions": 588, "aqua": 410, "aromaplanning": 192, "arsians": 627, "art-vr": 361, "asari-vr": 148, "asian-naughty-team": 726, "asiansexdiaryvr": 219, "asiansexvr": 629,
	"astrodomina": 520, "astrodominavr": 404, "avervr": 369, "baberoticavr": 247, "babykxtten": 321, "backalleytoonz-vr": 645, "badfamilypov": 433, "bakunouvr": 339, "balkan-girls": 655, "beatstrokervr": 626, "best-of": 541, "bestporn": 659, "big-ass-latinavr": 444,
	"blazed-studios-vr": 276, "bliss-vr": 274, "blondefoxsilverfoxvr": 805, "blondehexe": 393, "blondehexe-time": 707, "blowjobnow": 489, "blush-erotica": 385, "bootyliciousmag": 911, "bosslesson": 801, "bravomodelsmedia": 81, "brazilpartyorgy": 506, "brilliantvr": 689,
	"bs-production": 386, "bukkakeorgy": 595, "burningangelvr": 67, "busty-lover": 791, "calvista": 928, "caribbeancom": 222, "casanova": 184, "catalina-cruz": 661, "catalina-cruz-vr": 638, "chickpass-amateurs": 319, "chickpass-vr": 320, "chinchinvr": 188,
	"christine-ash": 625, "ck-studios": 734, "clubseventeenvr": 40, "clubsweethearts": 941, "clubsweethearts-legacy": 287, "cobrapmv": 683, "concoxxxion": 802, "concoxxxion-vr": 753, "cosmoplanetsvr": 252, "covertjapan": 221, "crazymouthmeat": 735, "cuck-hunter": 745,
	"cuckold-vr": 522, "cuckoldest": 935, "cuddle-mocap": 584, "cumcoders": 212, "cuminvr": 560, "cumpilations-studio": 479, "cumrotic": 662, "cupacakeus": 480, "curatedvr": 580, "czechar": 551, "czechvr": 17, "dama-movies": 589, "dandy": 355, "dans-vr": 776,
	"darlo-entertainment": 731, "darya-jane": 624, "ddfnetworkvr": 34, "deepinsex": 266, "deepinsex2d": 681, "desire-reels-by-amp": 751, "desire-room": 812, "desire-room-render": 953, "deviantsvr": 351, "dirty-cinema": 742, "domingo-network": 709, "dreamcam": 186,
	"dreamdollsvr": 697, "dreamticket": 746, "dreamticketvr": 119, "dynamiceyes": 263, "ebony-vr-solos": 387, "ebonyvr": 152, "ellielouisevr": 265, "emilybloom": 42, "emilys-art": 646, "enjoyit": 62, "epicvr": 611, "erikalust": 125, "erin-electra": 446, "erocomtv": 649,
	"erotic-sinners": 349, "erotimevr": 208, "ethernalvr": 620, "eve-sweets-playground": 447, "evrproductions": 535, "fakingsvr": 56, "familyhookups": 923, "familyscrew": 944, "familyscrew-legacy": 701, "fantasista-vr": 389, "fantastica": 197, "fantasyvr-studio": 671,
	"fap-vr": 610, "fapp3d": 130, "fatp": 382, "fbomb-studioz": 407, "feelmevr": 471, "femdomfantasyvr": 497, "fetish-maniacs": 794, "fetmagic": 736, "ffstockings": 63, "firstanalteens": 507, "fitloversvr": 658, "flat-desire-room": 896, "fleshy-body-vr": 397, "flexidolls": 508,
	"flowpov": 532, "foot-glamour": 733, "footfetishnetwork": 755, "footsiebay": 425, "forbidden": 537, "frameleaks": 851, "fsknightsvisual": 341, "fuck-a-fan": 771, "fuckonstreet": 593, "fuckpassvr": 352, "fucktruck-vr": 398, "full-service-pov": 857, "g4f": 637, "gasvr": 430,
	"get-your-fix": 959, "girlsway": 715, "glamour-vr": 440, "glosstightsglamourvr": 337, "good-porn": 825, "grandmams-legacy": 289, "grandparentsx": 946, "grandparentsx-legacy": 699, "grannies-vr": 214, "hamezo": 600, "heathering": 350, "hentaivr": 120, "herfirstvr": 231,
	"herpassionvr": 882, "holivr": 49, "hologirlsvr": 22, "homegrown-video": 951, "hookfer": 240, "hookup-hotshot": 748, "horny-household": 803, "hot-babes-vr": 374, "ichimonkai": 318, "iloveitshiny": 259, "immersex": 527, "immoral-pov": 774, "immoral-productions": 773,
	"immoralfamily": 775, "immorallive": 310, "innocent-holes": 793, "interracial-bangers": 792, "intima2d": 704, "intimacy-shots": 903, "intimavr": 573, "intimovr": 264, "isex": 654, "istripper": 171, "italianfetishvr": 465, "jackandjillvr": 367, "jackson-stock": 579,
	"japan-lust": 759, "japornxxx": 819, "jimmydraws": 173, "jj": 692, "johntron-vr": 333, "johntronbargirls": 651, "johntronx": 677, "joi-babes": 756, "joymii": 920, "justvr": 126, "jvrporn": 75, "kesha-ortega": 908, "kgb-the-travel-cam": 667, "khloesweets": 888,
	"kima-sonoma": 789, "kink-vr": 575, "kinky-spa": 922, "kinkygirlsberlin": 342, "kinkykingdom": 478, "kira-queen": 906, "kittymocap": 693, "kmpvr": 209, "koalavr": 189, "la-production": 583, "ladyparfume": 604, "lederhosengangbang": 509, "legsex": 910, "leninacrowne": 223,
	"les-worship": 761, "lethalhardcore": 505, "lethalhardcorevr": 159, "letsgobi": 945, "lewd-fraggy": 603, "lexis-star": 722, "lezvr": 76, "libertinevr": 104, "lightsouthern": 90, "lilly-bella-vr": 571, "little-dragon": 855, "littlecapricevr": 218, "livie-b-vr": 673,
	"loveherfeetvr": 260, "luke-cooper": 875, "luminous": 694, "lust-labz": 737, "lustreality": 233, "lustyvr": 371, "majamagic": 165, "mannys": 428, "mannysx": 559, "markohs": 149, "marrionvr": 139, "masqueradeporn": 606, "maturepickup": 644, "maturesvr": 314,
	"maturetofuck": 878, "max-a": 198, "max-pleasure": 955, "maximo-garcia": 893, "maxing": 345, "mayashandjobsvr": 93, "meangirlsvr": 60, "medfetvr": 498, "melina-may": 539, "melissa-stratton-presents": 811, "micky-muffin-vr": 891, "mike-and-kate": 309, "milf-trip": 948,
	"milfuckd": 763, "milfvr": 69, "miluvr": 140, "mindbender": 740, "mindbodylust": 254, "miss-lexa": 666, "miss-lexa-vr": 634, "mmm100": 106, "modelmedia-vr": 456, "mondelde-vr": 359, "monika-foxxx-studio": 889, "mousouvr": 317, "mrmichiru": 307, "mugon-vr": 344,
	"mugur-porn-vr": 375, "mugurs-world": 711, "muscleworshipvr": 347, "mutiny-vr": 401, "my-big-black-dick-pov": 860, "my-pov-fam": 766, "mybangvan": 510, "myschoollife": 721, "mysweetapple": 682, "mysweetapplevr": 657, "myvrsin": 115, "namyvr": 880, "nathans-sluts": 859,
	"natural-high": 331, "naughtyfreyavr": 210, "naughtyjoi": 663, "neonhappi": 685, "nesty": 937, "net69vr": 48, "never-lonely-vr": 353, "new-porn-starlets": 622, "nick-ross-vr": 640, "no2studiovr": 257, "noir": 400, "nucosplay": 729, "nude-yoga-porn": 757, "nuriamillanvr": 674,
	"nylon-xtreme": 227, "office-ks-vr": 248, "okfoxy": 490, "olipsvr": 144, "only-vr": 415, "only3x": 710, "only3xvr": 183, "onlytaboo": 511, "onlytease": 201, "orenovr": 525, "p-box-vr": 250, "parellalwordle": 196, "passions-only": 767, "peeping-thom": 303,
	"pegasproductions": 66, "pervertclips": 512, "perverted-pov": 765, "pervrt": 85, "peters-kingdom": 762, "petersmax": 137, "petersprimovr": 135, "pig-bros-productions": 586, "pinko": 850, "pip-vr": 246, "playhard-vr": 641, "plushies-tv": 708, "plushiesvr": 122, "pmg-girls": 703,
	"pmv-lab": 676, "polyamorious-pov": 521, "pornbcn": 163, "porncornvr": 502, "pornforce": 434, "porno-dan": 772, "pornodanvr": 278, "pornstarplatinum": 779, "pornworld": 741, "pov-mania": 856, "pov-masters": 758, "povcentral": 529, "povcentralvr": 193, "povmovies": 513,
	"prestige": 302, "private-jet": 455, "private-place": 553, "propertysex": 286, "propertysexvr": 46, "ps-porn": 653, "ps-porn-vr": 256, "pure-taboo": 716, "purevrsite": 887, "purityvr": 454, "putalocura": 504, "putalocuravr": 423, "pvrstudio": 199, "r3dvr": 939, "radical": 275,
	"raw-white-meat": 764, "real-voyeur": 597, "realhotvr": 160, "realitylovers": 27, "realjamvr": 53, "realstoryvr": 591, "red-light-district-vr": 739, "red-star-berlin-vr": 368, "redheatvr": 842, "reevr": 107, "renderpleasure": 530, "rezowoods": 170, "rocket": 357,
	"rome-major": 827, "rosina-lux": 904, "roxysdream": 607, "sabdeluxe": 566, "sabdeluxe-foot-fetish": 706, "salome-prologue": 251, "scoop-vr": 262, "screamvr": 500, "screwboxvr": 124, "seducevr": 228, "seductive-vr": 561, "sensesvrporn": 281, "sexbabesvr": 51, "sexflexvideo": 514,
	"sexycuckold": 515, "sh-studios-2": 938, "shadow-dimitri": 905, "shalina-devine-productions": 461, "shaundamxxx": 705, "she-seduced-me": 752, "she-will-cheat": 925, "sheer-experience": 206, "shinaryen": 747, "shinyvideos": 392, "silk-labo": 582, "simplyanal": 325, "sinbros": 743,
	"sinfulxxx": 942, "sinfulxxx-legacy": 288, "sinners-vault-vr": 804, "sinsdealers": 98, "sinsvr": 112, "slipperymassage": 516, "slowmotion": 636, "slr-avp": 587, "slr-cg-labs": 834, "slr-for-women": 672, "slr-labs": 383, "slr-originals": 224, "slr-originals-bts": 340,
	"slr-outtakes": 531, "sluts-around-town": 768, "sluts-of-spain": 550, "sodcreate": 216, "sophiasvr": 664, "spicy-vr": 443, "splinevr": 373, "squeeze-vr": 285, "squirt-god": 777, "ssr-vr": 841, "staminatraining": 567, "starfkrs": 962, "stasyq": 713, "stasyqvr": 79,
	"steel-vr": 452, "stockingsvr": 47, "stockingvideos": 714, "strictlyglamourvr": 280, "stripzvr": 108, "suckmevr": 403, "summerhart": 405, "summersinnerscom": 702, "swallowbay": 298, "sweetlonglips": 809, "sweetlonglipsvr": 177, "tabithaxxx": 799, "taboo-vr-porn": 354,
	"taboovr": 346, "tadpole": 732, "tadpolexxxstudio": 217, "teasetime": 549, "teddy-tarantino": 864, "teddy-tarantino-8k": 940, "teens-like-it-big": 862, "tenshigao": 728, "teppanvr": 194, "test-qa": 712, "texas-and-blondes": 907, "thai-goddess": 881, "thatrandomeditor": 414,
	"thedinodidit": 876, "thehotwifeza": 698, "thehotwifeza-vr": 396, "thelockedcockchronicles": 116, "thevirtualpornwebsite": 68, "third-base": 475, "throattlevr": 642, "tmavr": 207, "tmwvrnet": 26, "tommystone": 680, "top-xxx": 470, "torbeamateur": 501, "touch-me": 458,
	"toysttv": 890, "trike-patrol": 949, "tuktuk-patrol": 950, "tuktukpatrol-vr": 899, "upclosevr": 565, "v1vr": 195, "vertex-love": 877, "vesuvianwoman": 964, "vexxy-bliss": 798, "video-team": 927, "vipsexvault": 954, "viro-playspace": 304, "virtual-papi": 381,
	"virtual-real-amateur": 71, "virtual-real-passion": 57, "virtual-real-porn": 2, "virtualexotica": 176, "virtualpassionvr": 656, "virtualpee": 99, "virtualporn360": 4, "virtualporndesire": 28, "virtualxporn": 54, "vixenvr": 19, "vr-amateur": 412, "vr-intimacy": 158,
	"vr-japanese-idols-party": 362, "vr-japanese-pornstars-stay-home-routines": 372, "vr-massage": 519, "vr-paradise": 814, "vr-passporn": 477, "vr-pornnow": 378, "vr-pornnow-2d": 872, "vr-pornnow-cgi": 528, "vr-queens": 377, "vr3000": 33, "vrallure": 213, "vranimeted": 138,
	"vrbuz": 235, "vrclubz": 86, "vrcucking": 472, "vrdinky": 581, "vrdome": 441, "vreal-18k": 599, "vredging": 245, "vrextasy": 203, "vrfootfetish": 232, "vrgfriend": 467, "vrgirlz": 846, "vrgoddess": 279, "vrhard": 449, "vrhotwife": 617, "vrhush": 64, "vrinasia": 558,
	"vrixxens": 332, "vrjjproductions": 234, "vrlab9division": 390, "vrlatina": 110, "vrmagic": 639, "vrmansion": 450, "vrmassaged": 556, "vrmodelphotography": 237, "vrmodels": 162, "vroomed": 380, "vrparadisexxx": 87, "vrpfilms": 97, "vrplayful": 495, "vrpornjack": 172,
	"vrpornox": 675, "vrsexperts": 89, "vrsexygirlz": 109, "vrsmokers": 348, "vrsolos": 182, "vrspy": 491, "vrstars": 271, "vrsun": 544, "vrteens": 74, "vrvids": 261, "waap-entertainment": 723, "waapvr": 117, "wallstreetbanker": 894, "wankitnow": 315, "wankitnowvr": 154,
	"wankzvr": 24, "wastelandvr": 88, "we-are-playful": 725, "wearecrazy": 684, "whorecraftvr": 121, "whoresinpublic": 594, "wildgangbangs": 517, "wow": 249, "xjellyfish": 161, "xvirtual": 123, "yanksvr": 37, "yellow-pinkman": 388, "yogavr": 466, "your-thai-girlfriend": 936,
	"zaawaadivr": 268, "zentai-fantasy": 255, "zexyvr": 157,
}
