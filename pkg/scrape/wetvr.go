package scrape

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

type wetVRRelease struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CachedSlug  string   `json:"cachedSlug"`
	ReleasedAt  string   `json:"releasedAt"`
	PosterUrl   string   `json:"posterUrl"`
	ThumbUrls   []string `json:"thumbUrls"`
	TrailerUrl  string   `json:"trailerUrl"`
	Actors      []struct {
		Name string `json:"name"`
	} `json:"actors"`
	DownloadOptions []struct {
		Quality  string `json:"quality"`
		Filename string `json:"filename"`
	} `json:"downloadOptions"`
}

// Scene:Duration map
var wetVRDurations = map[int]int{75734: 2820, 75701: 3120, 75677: 2580, 75646: 2160, 75625: 1860, 75594: 2580, 75568: 3240, 75540: 2520, 75515: 2580, 75486: 2760, 75445: 2460, 75427: 2160, 75368: 2040, 75333: 2400, 75289: 2100, 75299: 3000, 75232: 2820, 75195: 2700, 75161: 2580, 75128: 2580, 75091: 2640, 75059: 2160, 75024: 2520, 74990: 2820, 74962: 2760, 74928: 2820, 74889: 2340, 74820: 2280, 74776: 2880, 74856: 3360, 74741: 3000, 74710: 2400, 74665: 2280, 74555: 2520, 74629: 2160, 74596: 2280, 69778: 3120, 74519: 1980, 74478: 2340, 74442: 2040, 74416: 3120, 74370: 2940, 74338: 2460, 74297: 2820, 74257: 3180, 74217: 3300, 74179: 2580, 69670: 3540, 74136: 3300, 74092: 2100, 74049: 2340, 74010: 2880, 73971: 2520, 73921: 2520, 73878: 2280, 73837: 2220, 73791: 2820, 69749: 3060, 73762: 2820, 73706: 2220, 73669: 3240, 73612: 2460, 73562: 3240, 73519: 3600, 73458: 2220, 73409: 3000, 73366: 2400, 73323: 3180, 73262: 2700, 73213: 1920, 73167: 3420, 69565: 2220, 69701: 2460, 69648: 1740, 69625: 2280, 69606: 2340, 69586: 1980, 69539: 2520, 69514: 2160, 69491: 2160, 69476: 3000, 69446: 2700, 69453: 2880, 69405: 1740, 69385: 2160, 69356: 2160, 69337: 1860, 69313: 3420, 69297: 1800, 69274: 2040, 69241: 2340, 69220: 2820, 69199: 2400, 69177: 2280, 69159: 1740, 69141: 2940, 69728: 2460, 69115: 2520, 69108: 3060, 69077: 2160, 69055: 2880, 69037: 2100, 69013: 2340, 68994: 3180, 68973: 2640, 68931: 2760, 68888: 2700, 68778: 2640, 68742: 2160, 68762: 3000, 68755: 2700, 68724: 2580, 68693: 3000, 68666: 3180, 68651: 2700, 68613: 2820, 68591: 2940, 68569: 3600, 68537: 2880, 68513: 2100, 68434: 2580, 68402: 2520, 68377: 2460, 68360: 1920, 68354: 2520, 68332: 2640, 68321: 2880, 68304: 2820, 68294: 2340, 68277: 2520, 68265: 2340, 68248: 2820, 68233: 2520, 68216: 2880, 68203: 2520, 68190: 2520, 68180: 2700, 68158: 2460, 68132: 2640, 68144: 2220, 68117: 1980, 68100: 2160, 68633: 2520, 68090: 2700, 68077: 1920, 67974: 2820, 67976: 2940, 67973: 2280, 67991: 2880, 67975: 2100, 67972: 2640, 67964: 2880, 67993: 2100, 67965: 2880, 68011: 2880, 67971: 1860, 67989: 1860, 68086: 2160, 68020: 2100}

type wetVRReleaseList struct {
	Items      []wetVRRelease `json:"items"`
	Pagination struct {
		NextPage   string `json:"nextPage"`
		TotalItems int    `json:"totalItems"`
		TotalPages int    `json:"totalPages"`
	} `json:"pagination"`
}

func fetchJSON(url string, target interface{}) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	httpConfig := GetCoreDomain(url) + "-scraper"
	log.Debugf("Using Header/Cookies from %s", httpConfig)
	SetupHtmlRequest(httpConfig, req)

	req.Header.Set("x-site", "wetvr.com")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse the JSON
	return json.Unmarshal(body, target)
}

func WetVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "wetvr"
	siteID := "WetVR"
	logScrapeStart(scraperID, siteID)

	processScene := func(scene wetVRRelease) {
		baseURL := "https://wetvr.com"
		sceneURL := fmt.Sprintf("%s/video/%s", baseURL, scene.CachedSlug)

		// Skip if scene already exists in database
		if funk.ContainsString(knownScenes, sceneURL) {
			return
		}

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "WetVR"
		sc.Site = siteID
		sc.Title = scene.Title
		sc.HomepageURL = sceneURL
		sc.MembersUrl = strings.Replace(sceneURL, baseURL+"/", baseURL+"/members/", 1)
		sc.SiteID = fmt.Sprintf("%d", scene.ID)
		sc.SceneID = slugify.Slugify(sc.Site + "-" + sc.SiteID)

		// Set duration from our mapping (convert seconds to minutes)
		if duration, ok := wetVRDurations[scene.ID]; ok {
			sc.Duration = duration / 60 // Integer division will round down
		}

		// Convert release date format
		if t, err := time.Parse(time.RFC3339, scene.ReleasedAt); err == nil {
			sc.Released = t.Format("2006-01-02")
		}
		sc.Synopsis = scene.Description

		// Cover image
		if scene.PosterUrl != "" {
			sc.Covers = append(sc.Covers, scene.PosterUrl)
		}

		// Gallery
		sc.Gallery = scene.ThumbUrls

		// Cast
		for _, actor := range scene.Actors {
			sc.Cast = append(sc.Cast, actor.Name)
		}

		// Trailer
		if scene.TrailerUrl != "" {
			sc.TrailerType = "url"
			sc.TrailerSrc = scene.TrailerUrl
		}

		// Filenames from downloadOptions
		for _, opt := range scene.DownloadOptions {
			sc.Filenames = append(sc.Filenames, opt.Filename)
		}

		out <- sc
	}

	if singleSceneURL != "" {
		// Extract slug from URL - handle both old and new URL formats
		slug := singleSceneURL
		if strings.Contains(singleSceneURL, "/video/") {
			slug = strings.TrimPrefix(singleSceneURL, "https://wetvr.com/video/")
		}
		// Remove any trailing slashes and query parameters
		slug = strings.Split(slug, "?")[0]
		slug = strings.TrimSuffix(slug, "/")
		// Get the last part of the path as the slug
		parts := strings.Split(slug, "/")
		slug = parts[len(parts)-1]

		apiURL := fmt.Sprintf("https://wetvr.com/api/releases/%s", slug)

		var scene wetVRRelease
		if err := fetchJSON(apiURL, &scene); err != nil {
			log.Error(err)
			return err
		}

		if scene.ID == 0 {
			log.Errorf("[%s] Failed to get valid scene data for %s", scraperID, apiURL)
			return fmt.Errorf("invalid scene data received")
		}

		processScene(scene)
	} else {
		page := 1
		sceneCount := 0

		for {
			apiURL := fmt.Sprintf("https://wetvr.com/api/releases?sort=latest&page=%d", page)
			// Skip per-page logging

			var releases wetVRReleaseList
			if err := fetchJSON(apiURL, &releases); err != nil {
				log.Error(err)
				return err
			}

			// Skip per-page scene count logging
			if len(releases.Items) == 0 {
				break
			}

			for _, scene := range releases.Items {
				// Hard rule: skip scene 75122 (duplicate of 75091)
				if scene.ID == 75122 {
					continue
				}
				if scene.ID != 0 {
					processScene(scene)
					sceneCount++
				}
			}

			if limitScraping {
				break
			}
			page++
		}

		log.Infof("[%s] Successfully scraped %d new scenes", scraperID, sceneCount)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("wetvr", "WetVR", "https://wetvr.com/images/sites/wetvr/wetvr-favicon.ico", "wetvr.com", WetVR)
}
