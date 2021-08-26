package scrape

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func VirtualRealPornSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, scraperID string, siteID string, URL string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("virtualrealporn.com", "virtualrealtrans.com", "virtualrealgay.com", "virtualrealpassion.com", "virtualrealamateurporn.com")
	siteCollector := createCollector("virtualrealporn.com", "virtualrealtrans.com", "virtualrealgay.com", "virtualrealpassion.com", "virtualrealamateurporn.com")
	castCollector := createCollector("virtualrealporn.com", "virtualrealtrans.com", "virtualrealgay.com", "virtualrealpassion.com", "virtualrealamateurporn.com")
	castCollector.AllowURLRevisit = true

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		sc.Studio = "VirtualRealPorn"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		var tmpCast []string

		// Scene ID - get from JavaScript
		e.ForEach(`script[id="virtualreal_video-streaming-js-extra"]`, func(id int, e *colly.HTMLElement) {
			var jsonObj map[string]interface{}
			jsonData := e.Text[strings.Index(e.Text, "{") : len(e.Text)-12]
			json.Unmarshal([]byte(jsonData), &jsonObj)

			sc.SiteID = jsonObj["vid"].(string)
			sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		})

		// Title
		e.ForEach(`title`, func(id int, e *colly.HTMLElement) {
			sc.Title = e.Text
			sc.Title = strings.TrimSpace(strings.Replace(sc.Title, "▷ ", "", -1))
			sc.Title = strings.TrimSpace(strings.Replace(sc.Title, fmt.Sprintf(" - %v.com", sc.Site), "", -1))
		})

		// Cover URLs
		e.ForEach(`meta[property="og:image"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				sc.Covers = append(sc.Covers, strings.Split(e.Request.AbsoluteURL(e.Attr("content")), "?")[0])
			}
		})

		// Gallery
		e.ForEach(`figure[itemprop="associatedMedia"] a`, func(id int, e *colly.HTMLElement) {
			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(strings.Split(e.Attr("href"), "?")[0]))
		})

		// Tags
		e.ForEach(`a[href*="/tag/"] span`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})
		if scraperID == "virtualrealgay" {
			sc.Tags = append(sc.Tags, "Gay")
		}

		// Duration / Release date / Synopsis
		e.ForEach(`script[type='application/ld+json'][class!='yoast-schema-graph']`, func(id int, e *colly.HTMLElement) {
			var jsonResult map[string]interface{}
			json.Unmarshal([]byte(e.Text), &jsonResult)

			duration := jsonResult["duration"].(string)
			tmpParts := strings.Split(duration, ":")
			if len(tmpParts) == 2 {
				sc.Duration, _ = strconv.Atoi(tmpParts[0])
			} else {
				tmpParts = strings.Split(duration, "h ")
				if len(tmpParts) == 2 {
					hours, _ := strconv.Atoi(tmpParts[0])
					minutes, _ := strconv.Atoi(tmpParts[1])
					sc.Duration = hours*60 + minutes
				}
			}

			sc.Released = jsonResult["datePublished"].(string)

			sc.Synopsis = html.UnescapeString(jsonResult["description"].(string))

			cast := jsonResult["actors"].([]interface{})
			for _, v := range cast {
				m := v.(map[string]interface{})
				tmpCast = append(tmpCast, e.Request.AbsoluteURL(m["url"].(string)))
			}
		})

		e.ForEach(`script[id="downloadLinks-js-extra"]`, func(id int, e *colly.HTMLElement) {
			if id == 0 {
				jsonData := e.Text[strings.Index(e.Text, "{") : len(e.Text)-12]
				fpName := gjson.Get(jsonData, "videopart").String()

				// A couple of pages will crash the scraper (ex. url https://virtualrealporn.com/&mode=streaming)
				if fpName == "" {
					return
				}

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
				outFilenames = append(outFilenames, siteIDAcronym+"_"+fpName+"_Trailer_PS4_180_sbs.mp4") // Trailer
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_3K_180_sbs.mp4")                 // PS4
				outFilenames = append(outFilenames, siteIDAcronym+"_"+fpName+"_180_sbs.mp4")             // PS4 (older videos)
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_180_sbs.mp4")            // PS4 (oldest videos)
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_4096x2040_180_sbs.mp4")          // PS4 Pro
				outFilenames = append(outFilenames, siteIDAcronym+"_"+fpName+"_Pro_180_sbs.mp4")         // PS4 Pro (older videos)
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_Pro_180_sbs.mp4")        // PS4 Pro (oldest videos)

				// Oculus Go / Quest
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_Trailer.mp4")       // Trailer (same for Oculus Rift (S) / Vive / Windows MR)
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_4864_180x180_3dh.mp4")      // 4K+
				outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_h264P_180x180_3dh.mp4") // 4K+ (older videos)
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_4K_30M_180x180_3dh.mp4")    // 4K HQ (same for Gear VR / Daydream and Oculus Rift (S) / Vive / Windows MR)
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_4K_h265_180x180_3dh.mp4")   // 4K h265 (same for Oculus Rift (S) / Vive / Windows MR)
				outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_vp9_180x180_3dh.mp4")   // 4K VP9 (older videos; same for Gear VR / Daydream and Oculus Rift (S) / Vive / Windows MR)
				outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_180x180_3dh.mp4")       // 4K h264 (older videos; same for Gear VR / Daydream)

				// Gear VR / Daydream
				outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_Trailer_Streaming_3dh.mp4") // Trailer
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_4K_180x180_3dh.mp4")            // 4K (same for Smartphone)

				// Smartphone
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_Trailer_-_Smartphone.mp4") // Trailer
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_1920_180x180_3dh.mp4")             // Full HD (same for Oculus Rift (S) / Vive / Windows MR)
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_1920.mp4")                 // Full HD (older videos; same for Oculus Rift (S) / Vive / Windows MR)

				// Oculus Rift (S) / Vive / Windows MR
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_5K_30M_180x180_3dh.mp4")     // 5K HQ
				outFilenames = append(outFilenames, siteID+"_"+fpName+"_5K_180x180_3dh.mp4")         // 5K
				outFilenames = append(outFilenames, siteID+"_-_"+fpName+"_-_5K_180x180_3dh.mp4")     // 5K (older videos)
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_5K_180x180_3dh.mp4") // 5K (before site update)
				outFilenames = append(outFilenames, siteID+".com_-_"+fpName+"_-_h264.mp4")           // 4K 264 (older videos)

				sc.Filenames = outFilenames
			}
		})

		ctx := colly.NewContext()
		ctx.Put("scene", &sc)

		for i := range tmpCast {
			castCollector.Request("GET", tmpCast[i], nil, ctx, nil)
		}

		out <- sc
	})

	castCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := e.Request.Ctx.GetAny("scene").(*models.ScrapedScene)

		var name string
		e.ForEach(`h1.model-title`, func(id int, e *colly.HTMLElement) {
			name = strings.TrimSpace(strings.Split(e.Text, " (")[0])
		})

		var gender string
		e.ForEach(`div.model-info div.one-half div`, func(id int, e *colly.HTMLElement) {
			if strings.Split(e.Text, " ")[0] == "Gender" {
				gender = strings.Split(e.Text, " ")[1]
			}
		})

		if gender == "Female" || gender == "Transgender" {
			sc.Cast = append(sc.Cast, name)
		} else if sc.SiteID == "VirtualRealGay" || sc.SiteID == "VirtualRealPassion" {
			sc.Cast = append(sc.Cast, name)
		}
	})

	siteCollector.OnHTML(`.searchBox option`, func(e *colly.HTMLElement) {
		pageURL := e.Request.AbsoluteURL(e.Attr("data-url"))
		siteCollector.Visit(pageURL)
	})

	siteCollector.OnHTML(`a.w-portfolio-item-anchor`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	if scraperID == "virtualrealamateur" {
		siteCollector.Visit(URL)
	} else if scraperID == "virtualrealgay" {
		siteCollector.Visit(URL + "porn-actor/")
	} else {
		siteCollector.Visit(URL + "porn-actress/")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func VirtualRealPorn(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, "virtualrealporn", "VirtualRealPorn", "https://virtualrealporn.com/")
}
func VirtualRealTrans(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, "virtualrealtrans", "VirtualRealTrans", "https://virtualrealtrans.com/")
}
func VirtualRealAmateur(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, "virtualrealamateur", "VirtualRealAmateurPorn", "https://virtualrealamateurporn.com/")
}
func VirtualRealGay(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, "virtualrealgay", "VirtualRealGay", "https://virtualrealgay.com/")
}
func VirtualRealPassion(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene) error {
	return VirtualRealPornSite(wg, updateSite, knownScenes, out, "virtualrealpassion", "VirtualRealPassion", "https://virtualrealpassion.com/")
}

func init() {
	registerScraper("virtualrealporn", "VirtualRealPorn", "https://pbs.twimg.com/profile_images/921297545195859968/E5-ClWkm_200x200.jpg", VirtualRealPorn)
	registerScraper("virtualrealtrans", "VirtualRealTrans", "https://pbs.twimg.com/profile_images/921298616970555392/3coTQ6UZ_200x200.jpg", VirtualRealTrans)
	registerScraper("virtualrealgay", "VirtualRealGay", "https://pbs.twimg.com/profile_images/921298132129992704/jIOE0LxX_200x200.jpg", VirtualRealGay)
	registerScraper("virtualrealpassion", "VirtualRealPassion", "https://pbs.twimg.com/profile_images/921298874249175041/LjWabMPh_200x200.jpg", VirtualRealPassion)
	registerScraper("virtualrealamateur", "VirtualRealAmateurPorn", "https://mcdn.vrporn.com/files/20170718094205/virtualrealameteur-vr-porn-studio-vrporn.com-virtual-reality.png", VirtualRealAmateur)
}
