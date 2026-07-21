package scrape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func VirtualRealPornSite(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)
	page := 1

	// Covers and gallery images are served from static.virtualrealhub.com (new site CDN)
	imageCollector := createCollector("virtualrealporn.com", "virtualrealtrans.com", "virtualrealgay.com", "virtualrealpassion.com", "virtualrealamateurporn.com", "static.virtualrealhub.com")
	sceneCollector := createCollector("virtualrealporn.com", "virtualrealtrans.com", "virtualrealgay.com", "virtualrealpassion.com", "virtualrealamateurporn.com")
	siteCollector := createCollector("virtualrealporn.com", "virtualrealtrans.com", "virtualrealgay.com", "virtualrealpassion.com", "virtualrealamateurporn.com")

	imageCollector.OnResponse(func(r *colly.Response) {
		if _, _, err := image.Decode(bytes.NewReader(r.Body)); err == nil {
			r.Ctx.Put("valid", "1")
		}
	})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "VR"
		sc.Studio = "VirtualRealPorn"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		// Scene ID - from dl8-video custom element (new site)
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.SiteID = e.Attr("data-video-id")
				sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
			}
		})

		// Title
		e.ForEach(`title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(strings.Split(e.Text, "|")[0])
			sc.Title = strings.TrimSpace(strings.Replace(sc.Title, "▷ ", "", -1))
		})

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if len(sc.Covers) == 0 {
				u := strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0]
				ctx := colly.NewContext()
				if err := imageCollector.Request("GET", u, nil, ctx, nil); err == nil {
					if ctx.Get("valid") != "" {
						sc.Covers = append(sc.Covers, u)
					}
				}
			}
		})

		// Gallery (screenshots grid) - full image is on the anchor href; <img> is lazy-loaded
		e.ForEach(`a.vd-screenshots__item`, func(id int, e *colly.HTMLElement) {
			u := e.Attr("href")
			if u == "" {
				u = e.Attr("data-gallery-src")
			}
			u = e.Request.AbsoluteURL(strings.Split(u, "?")[0])
			if u != "" {
				sc.Gallery = append(sc.Gallery, u)
			}
		})

		// Tags
		e.ForEach(`a.vd-tags__tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})
		if scraperID == "virtualrealgay" {
			sc.Tags = append(sc.Tags, "Gay")
		}

		// Cast - from pornstar sections (new site)
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`div.vd-pornstar`, func(id int, e *colly.HTMLElement) {
			name := strings.TrimSpace(e.DOM.Find("span.vd-pornstar__name").Text())
			profileURL := e.Request.AbsoluteURL(e.DOM.Find("a.vd-pornstar__link").AttrOr("href", ""))
			if name != "" {
				sc.Cast = append(sc.Cast, name)
				sc.ActorDetails[name] = models.ActorDetails{Source: scraperID + " scrape", ProfileUrl: profileURL}
			}
		})

		// Duration / Release date / Synopsis from JSON-LD VideoObject
		e.ForEach(`script[type='application/ld+json']`, func(id int, e *colly.HTMLElement) {
			var jsonResult map[string]interface{}
			if err := json.Unmarshal([]byte(e.Text), &jsonResult); err != nil {
				return
			}
			if jsonResult["@type"] != "VideoObject" {
				return
			}

			// Duration: ISO 8601 e.g. "PT2406S" (total seconds)
			if dur, ok := jsonResult["duration"].(string); ok {
				dur = strings.TrimPrefix(dur, "PT")
				dur = strings.TrimSuffix(dur, "S")
				if secs, err := strconv.Atoi(dur); err == nil {
					sc.Duration = secs / 60
				}
			}

			// Release date: "uploadDate" ISO format, take date part only
			if uploaded, ok := jsonResult["uploadDate"].(string); ok && len(uploaded) >= 10 {
				sc.Released = uploaded[:10]
			}

			// Synopsis
			if desc, ok := jsonResult["description"].(string); ok {
				sc.Synopsis = desc
			}
		})

		// Download filenames - derive fpName from URL slug (new site no longer exposes JS variable)
		urlParts := strings.Split(strings.TrimSuffix(sc.HomepageURL, "/"), "/")
		fpName := urlParts[len(urlParts)-1]

		if fpName != "" {
			siteIDAcronym := "VRP"
			if siteID == "VirtualRealTrans" {
				siteIDAcronym = "VRT"
			}
			if siteID == "VirtualRealAmateurPorn" {
				siteIDAcronym = "VRAM"
			}
			if siteID == "VirtualRealGay" {
				siteIDAcronym = "VRG"
			}
			if siteID == "VirtualRealPassion" {
				siteIDAcronym = "VRPA"
			}

			var outFilenames []string

			// Playstation VR
			outFilenames = append(outFilenames, siteIDAcronym+"_"+fpName+"_Trailer_PS4_180_sbs.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_3K_180_sbs.mp4")
			outFilenames = append(outFilenames, siteIDAcronym+"_"+fpName+"_180_sbs.mp4")
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_180_sbs.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_4096x2040_180_sbs.mp4")
			outFilenames = append(outFilenames, siteIDAcronym+"_"+fpName+"_Pro_180_sbs.mp4")
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_Pro_180_sbs.mp4")

			// Oculus Go / Quest
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_Trailer.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_4864_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_h264P_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_4K_30M_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_4K_h265_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_vp9_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_180x180_3dh.mp4")

			// Gear VR / Daydream
			outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_Trailer_Streaming_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_4K_180x180_3dh.mp4")

			// Smartphone
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_Trailer_-_Smartphone.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_1920_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_1920.mp4")

			// Oculus Rift (S) / Vive / Windows MR
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_8K_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_5K_30M_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_"+fpName+"_5K_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_5K_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_5K_180x180_3dh.mp4")
			outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_h264.mp4")

			sc.Filenames = outFilenames
		}

		// Trailer setup
		params := models.TrailerScrape{
			SceneUrl:       sc.HomepageURL,
			HtmlElement:    `script[type="application/ld+json"]`,
			ContentPath:    "trailer.contentUrl",
			QualityPath:    "trailer.videoQuality",
			ContentBaseUrl: URL,
		}
		tmp, _ := json.Marshal(params)
		sc.TrailerType = "scrape_json"
		sc.TrailerSrc = string(tmp)

		if sc.SceneID != "" {
			out <- sc
		}
	})

	// Scene listing - new site uses a.data-title links, paginated via ?page=N on /videos/ path
	siteCollector.OnHTML(`a.data-title`, func(e *colly.HTMLElement) {
		sceneURL := strings.Split(e.Request.AbsoluteURL(e.Attr("href")), "?")[0]

		// On first scene of each page, queue next page (before visiting scenes)
		if e.Index == 0 && !limitScraping {
			page++
			siteCollector.Visit(fmt.Sprintf("%svideos/?page=%v", URL, page))
		}

		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit(fmt.Sprintf("%svideos/?page=%v", URL, page))
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VirtualRealPorn(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, singleSceneURL, "virtualrealporn", "VirtualRealPorn", "https://virtualrealporn.com/", singeScrapeAdditionalInfo, limitScraping)
}
func VirtualRealTrans(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, singleSceneURL, "virtualrealtrans", "VirtualRealTrans", "https://virtualrealtrans.com/", singeScrapeAdditionalInfo, limitScraping)
}
func VirtualRealAmateur(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, singleSceneURL, "virtualrealamateur", "VirtualRealAmateurPorn", "https://virtualrealamateurporn.com/", singeScrapeAdditionalInfo, limitScraping)
}
func VirtualRealGay(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, singleSceneURL, "virtualrealgay", "VirtualRealGay", "https://virtualrealgay.com/", singeScrapeAdditionalInfo, limitScraping)
}
func VirtualRealPassion(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, singleSceneURL, "virtualrealpassion", "VirtualRealPassion", "https://virtualrealpassion.com/", singeScrapeAdditionalInfo, limitScraping)
}

func init() {
	registerScraper("virtualrealporn", "VirtualRealPorn", "https://pbs.twimg.com/profile_images/921297545195859968/E5-ClWkm_200x200.jpg", "virtualrealporn.com", VirtualRealPorn)
	registerScraper("virtualrealtrans", "VirtualRealTrans", "https://pbs.twimg.com/profile_images/921298616970555392/3coTQ6UZ_200x200.jpg", "virtualrealtrans.com", VirtualRealTrans)
	registerScraper("virtualrealgay", "VirtualRealGay", "https://pbs.twimg.com/profile_images/921298132129992704/jIOE0LxX_200x200.jpg", "virtualrealgay.com", VirtualRealGay)
	registerScraper("virtualrealpassion", "VirtualRealPassion", "https://pbs.twimg.com/profile_images/921298874249175041/LjWabMPh_200x200.jpg", "virtualrealpassion.com", VirtualRealPassion)
	registerScraper("virtualrealamateur", "VirtualRealAmateurPorn", "https://mcdn.vrporn.com/files/20170718094205/virtualrealameteur-vr-porn-studio-vrporn.com-virtual-reality.png", "virtualrealamateurporn.com", VirtualRealAmateur)
}
