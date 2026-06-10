package scrape

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

// The VRBangers network no longer serves scene listings or scene pages to
// logged-out visitors — every HTML page 302-redirects to /join-now/. Their
// public content API (content.<site>.com/api/content/v1/videos) still works
// without authentication, so this scraper uses it for both discovery and
// scene metadata.

var vrbangersTagRE = regexp.MustCompile(`<[^>]*>`)

func VRBangersSite(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	// https://vrbangers.com/ -> https://content.vrbangers.com/
	contentURL := strings.Replace(URL, "//", "//content.", 1)
	apiBase := contentURL + "api/content/v1/videos"

	client := resty.New()
	client.SetHeader("User-Agent", UserAgent)
	if config.Config.Advanced.ScraperProxy != "" {
		log.Infof("Using proxy for scraping: %s.", config.Config.Advanced.ScraperProxy)
		client.SetProxy(config.Config.Advanced.ScraperProxy)
	}

	scrapeScene := func(slug string) {
		sceneAPIURL := apiBase + "/" + slug
		log.Infoln("visiting", sceneAPIURL)
		r, err := client.R().Get(sceneAPIURL)
		if err != nil {
			log.Errorf("Error visiting %s %s", sceneAPIURL, err)
			return
		}
		JsonMetadata := r.String()

		//if not valid scene...
		if gjson.Get(JsonMetadata, "status.message").String() != "Ok" {
			return
		}
		item := gjson.Get(JsonMetadata, "data.item")

		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = siteID
		sc.HomepageURL = URL + "video/" + slug + "/"

		//Scene ID - back 8 of the "id" via api response
		id := item.Get("id").String()
		if len(id) <= 15 {
			return
		}
		sc.SiteID = strings.TrimSpace(id[15:])
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(item.Get("title").String())

		// Filenames - VRBANGERS_the_missing_kitten_8K_180x180_3dh.mp4
		baseName := sc.Site + "_" + strings.TrimSpace(item.Get("videoSettings.videoShortName").String()) + "_"
		filenames := []string{"8K_180x180_3dh", "6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

		for i := range filenames {
			filenames[i] = baseName + filenames[i] + ".mp4"
		}

		sc.Filenames = filenames

		// Date - from API publish timestamp
		if publishedAt := item.Get("publishedAt").Int(); publishedAt > 0 {
			sc.Released = time.Unix(publishedAt, 0).UTC().Format("2006-01-02")
		}

		// Duration from API (seconds to minutes)
		apiDuration := item.Get("videoSettings.duration").Int()
		if apiDuration > 0 {
			sc.Duration = int(apiDuration / 60)
		}

		// Cover URLs
		for _, key := range []string{"poster.permalink", "heroImg.permalink"} {
			if permalink := item.Get(key).String(); permalink != "" {
				sc.Covers = append(sc.Covers, strings.Replace(contentURL+permalink, ".com//", ".com/", 1))
			}
		}

		// Gallery - https://content.vrbangers.com/uploads/2021/08/611b4e0ca5c54351494706_XL.jpg
		gallerytmp := item.Get("galleryImages.#.previews.#(sizeAlias==XL).permalink")
		for _, v := range gallerytmp.Array() {
			sc.Gallery = append(sc.Gallery, strings.Replace(contentURL+v.Str, ".com//", ".com/", 1))
		}

		// Synopsis - description is HTML, strip tags
		sc.Synopsis = html.UnescapeString(vrbangersTagRE.ReplaceAllString(item.Get("description").String(), " "))
		sc.Synopsis = strings.Join(strings.Fields(sc.Synopsis), " ")

		// Tags
		ignoreTags := []string{"180 vr", "360 vr", "4k vr porn", "5k vr porn", "6k vr porn", "8k vr porn", "12k vr porn"}
		for _, c := range item.Get("categories.#.name").Array() {
			tag := strings.ToLower(strings.TrimSpace(c.String()))
			if tag == "" || funk.ContainsString(ignoreTags, tag) {
				continue
			}
			sc.Tags = append(sc.Tags, tag)
		}
		if scraperID == "vrbgay" {
			sc.Tags = append(sc.Tags, "Gay")
		}

		// setup trailers
		if scraperID != "vrconk" {
			sc.TrailerType = "load_json"
			params := models.TrailerScrape{SceneUrl: sceneAPIURL, RecordPath: "data.item.videoPlayerSources.trailer", ContentPath: "src", QualityPath: "quality"}
			strParma, _ := json.Marshal(params)
			sc.TrailerSrc = string(strParma)
		}

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		for _, m := range item.Get("models").Array() {
			name := strings.TrimSpace(strings.Replace(m.Get("title").String(), ",", "", -1))
			if name == "" {
				continue
			}
			sc.Cast = append(sc.Cast, name)
			sc.ActorDetails[name] = models.ActorDetails{Source: scraperID + " scrape", ProfileUrl: URL + "model/" + m.Get("slug").String() + "/"}
		}

		out <- sc
	}

	if singleSceneURL != "" {
		// e.g. https://vrbangers.com/video/<slug>/
		parts := strings.Split(strings.Trim(strings.Split(singleSceneURL, "?")[0], "/"), "/")
		slug := parts[len(parts)-1]
		scrapeScene(slug)
	} else {
		page := 1
		totalPages := 1
		for page <= totalPages {
			listURL := fmt.Sprintf("%s?limit=50&page=%d&sort=latest", apiBase, page)
			log.Infoln("visiting", listURL)
			r, err := client.R().Get(listURL)
			if err != nil {
				log.Errorf("Error visiting %s %s", listURL, err)
				break
			}
			listJSON := r.String()
			if gjson.Get(listJSON, "status.message").String() != "Ok" {
				log.Errorf("Invalid response from %s: %s", listURL, gjson.Get(listJSON, "status.message").String())
				break
			}
			if tp := gjson.Get(listJSON, "data.pages").Int(); tp > 0 {
				totalPages = int(tp)
			}
			items := gjson.Get(listJSON, "data.items").Array()
			if len(items) == 0 {
				break
			}
			for _, listItem := range items {
				slug := listItem.Get("slug").String()
				if slug == "" {
					continue
				}
				sceneURL := URL + "video/" + slug + "/"
				if !funk.ContainsString(knownScenes, sceneURL) {
					scrapeScene(slug)
					// be polite to the API
					time.Sleep(time.Second)
				}
			}
			if limitScraping {
				break
			}
			page++
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VRBangers(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrbangers", "VRBangers", "https://vrbangers.com/", limitScraping)
}
func VRBTrans(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrbtrans", "VRBTrans", "https://vrbtrans.com/", limitScraping)
}
func VRBGay(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrbgay", "VRBGay", "https://vrbgay.com/", limitScraping)
}
func VRConk(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "vrconk", "VRCONK", "https://vrconk.com/", limitScraping)
}
func BlowVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "blowvr", "BlowVR", "https://blowvr.com/", limitScraping)
}
func ARPorn(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, singleSceneURL, "arporn", "ARPorn", "https://arporn.com/", limitScraping)
}

func init() {
	registerScraper("vrbangers", "VRBangers", "https://vrbangers.com/favicon/apple-touch-icon-144x144.png", "vrbangers.com", VRBangers)
	registerScraper("vrbtrans", "VRBTrans", "https://vrbtrans.com/favicon/apple-touch-icon-144x144.png", "vrbtrans.com", VRBTrans)
	registerScraper("vrbgay", "VRBGay", "https://vrbgay.com/favicon/apple-touch-icon-144x144.png", "vrbgay.com", VRBGay)
	registerScraper("vrconk", "VRCONK", "https://vrconk.com/favicon/apple-touch-icon-144x144.png", "vrconk.com", VRConk)
	registerScraper("blowvr", "BlowVR", "https://blowvr.com/favicon/apple-touch-icon-144x144.png", "blowvr.com", BlowVR)
	registerScraper("arporn", "ARPorn", "https://arporn.com/favicon/apple-touch-icon-144x144.png", "arporn.com", ARPorn)
}
