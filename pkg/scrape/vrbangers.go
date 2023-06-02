package scrape

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRBangersSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("vrbangers.com", "vrbtrans.com", "vrbgay.com", "vrconk.com", "blowvr.com")
	siteCollector := createCollector("vrbangers.com", "vrbtrans.com", "vrbgay.com", "vrconk.com", "blowvr.com")
	ajaxCollector := createCollector("vrbangers.com", "vrbtrans.com", "vrbgay.com", "vrconk.com", "blowvr.com")
	ajaxCollector.CacheDir = ""

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VRBangers"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		content_id := strings.Split(strings.Replace(sc.HomepageURL, "//", "/", -1), "/")[3]

		//https://content.vrbangers.com
		contentURL := strings.Replace(URL, "//", "//content.", 1)

		r, _ := resty.New().R().
			SetHeader("User-Agent", UserAgent).
			Get("https://content." + sc.Site + ".com/api/content/v1/videos/" + content_id)

		JsonMetadata := r.String()

		// Sneak Peek 2020 VRBangers URL will crash the scraper
		//if not valid scene...
		if gjson.Get(JsonMetadata, "status.message").String() != "Ok" {
			return
		}

		//Scene ID - back 8 of the"id" via api response
		sc.SiteID = strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.id").String()[15:])
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.title").String())

		// Filenames - VRBANGERS_the_missing_kitten_8K_180x180_3dh.mp4
		baseName := sc.Site + "_" + strings.TrimSpace(gjson.Get(JsonMetadata, "data.item.videoSettings.videoShortName").String()) + "_"
		filenames := []string{"8K_180x180_3dh", "6K_180x180_3dh", "5K_180x180_3dh", "4K_180x180_3dh", "HD_180x180_3dh", "HQ_180x180_3dh", "PSVRHQ_180x180_3dh", "UHD_180x180_3dh", "PSVRHQ_180_sbs", "PSVR_mono", "HQ_mono360", "HD_mono360", "PSVRHQ_ou", "UHD_3dv", "HD_3dv", "HQ_3dv"}

		for i := range filenames {
			filenames[i] = baseName + filenames[i] + ".mp4"
		}

		sc.Filenames = filenames

		var durationParts []string
		// Date & Duration
		e.ForEach(`div.single-video-info__list-item`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, ":")
			if len(parts) > 1 {
				switch strings.TrimSpace(parts[0]) {
				case "Release date":
					tmpDate, _ := goment.New(strings.TrimSpace(parts[1]), "MMM D, YYYY")
					sc.Released = tmpDate.Format("YYYY-MM-DD")
				case "Duration":
					durationParts = strings.Split(strings.TrimSpace(parts[1]), " ")
					tmpDuration, err := strconv.Atoi(durationParts[0])
					if err == nil {
						sc.Duration = tmpDuration
					}
				}
			}

		})

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			tmpCover := strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0]
			if tmpCover != "https://vrbangers.com/wp-content/uploads/2020/03/VR-Bangers-Logo.jpg" && tmpCover != "https://vrbgay.com/wp-content/uploads/2020/03/VRB-Gay-Logo.jpg" && tmpCover != "https://vrbtrans.com/wp-content/uploads/2020/03/VRB-Trans-Logo.jpg" {
				sc.Covers = append(sc.Covers, tmpCover)
			}
		})

		sc.Covers = append(sc.Covers, e.ChildAttr(`section.static-banner__wrap picture img`, "src"))
		sc.Covers = append(sc.Covers, e.ChildAttr(`div.single-video-poster__preview img`, "src"))

		// Gallery - https://content.vrbangers.com/uploads/2021/08/611b4e0ca5c54351494706_XL.jpg
		gallerytmp := gjson.Get(JsonMetadata, "data.item.galleryImages.#.previews.#(sizeAlias==XL).permalink")
		for _, v := range gallerytmp.Array() {
			sc.Gallery = append(sc.Gallery, strings.Replace(contentURL+v.Str, ".com//", ".com/", 1))
		}

		// Synopsis
		sc.Synopsis = strings.TrimSpace(strings.Replace(e.ChildText(`div.single-video-description div.toggle-content__text`), `arrow_drop_up`, ``, -1))

		// Tags
		ignoreTags := []string{"180 vr", "6k vr porn", "8k vr porn", "4k vr porn"}
		e.ForEach(`a.single-video-categories__item`, func(id int, e *colly.HTMLElement) {
			tag := strings.ToLower(strings.TrimSpace(e.Text))
			for _, v := range ignoreTags {
				if tag == v {
					return
				}
			}
			sc.Tags = append(sc.Tags, tag)
		})
		e.ForEach(`div.single-video-info__positions span.video-positions__name`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})
		if scraperID == "vrbgay" {
			sc.Tags = append(sc.Tags, "Gay")
		}

		// setup trailers
		if scraperID != "vrconk" {
			sc.TrailerType = "load_json"
			params := models.TrailerScrape{SceneUrl: "https://content." + sc.Site + ".com/api/content/v1/videos/" + content_id, RecordPath: "data.item.videoPlayerSources.trailer", ContentPath: "src", QualityPath: "quality"}
			strParma, _ := json.Marshal(params)
			sc.TrailerSrc = string(strParma)
		}

		// Cast
		e.ForEach(`.single-video-info__content a[href^="/model/"]`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(strings.Replace(e.Text, ",", "", -1)))
		})

		out <- sc
	})

	siteCollector.OnHTML(`div.grid-video-item__title a`, func(e *colly.HTMLElement) {
		url := strings.Split(e.Attr("href"), "?")[0]
		sceneURL := e.Request.AbsoluteURL(url)

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`a.page-pagination__next`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("href"))

		siteCollector.Visit(pageURL)
	})

	siteCollector.Visit(URL + "videos/?sort=latest")

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VRBangers(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbangers", "VRBangers", "https://vrbangers.com/")
}
func VRBTrans(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbtrans", "VRBTrans", "https://vrbtrans.com/")
}
func VRBGay(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrbgay", "VRBGay", "https://vrbgay.com/")
}
func VRConk(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "vrconk", "VRCONK", "https://vrconk.com/")
}
func BlowVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VRBangersSite(wg, updateSite, knownScenes, out, "blowvr", "BlowVR", "https://blowvr.com/")
}

func init() {
	registerScraper("vrbangers", "VRBangers", "https://vrbangers.com/favicon/apple-touch-icon-144x144.png", VRBangers)
	registerScraper("vrbtrans", "VRBTrans", "https://vrbtrans.com/favicon/apple-touch-icon-144x144.png", VRBTrans)
	registerScraper("vrbgay", "VRBGay", "https://vrbgay.com/favicon/apple-touch-icon-144x144.png", VRBGay)
	registerScraper("vrconk", "VRCONK", "https://vrconk.com/favicon/apple-touch-icon-144x144.png", VRConk)
	registerScraper("blowvr", "BlowVR", "https://blowvr.com/favicon/apple-touch-icon-144x144.png", BlowVR)
}
