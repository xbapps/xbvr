package scrape

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func VRPorn(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, company string, siteURL string, singeScrapeAdditionalInfo string, limitScraping bool, masterSiteId string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)

	apiCollector := createCollector("vrporn.com")

	page := 1
	apiCollector.OnResponse(func(r *colly.Response) {
		jsonData := gjson.ParseBytes(r.Body)

		processScene := func(scene gjson.Result) {
			sc := models.ScrapedScene{}
			sc.SiteID = scene.Get("id").String()
			slug := scene.Get("slug").String()
			sc.SceneID = "vrporn-" + sc.SiteID

			sc.ScraperID = scraperID
			sc.SceneType = "VR"
			sc.Studio = company
			sc.Site = siteID
			sc.HomepageURL = "https://vrporn.com/" + slug + "/"

			sc.MasterSiteId = masterSiteId

			if scraperID == "" {
				// there maybe no site/studio if user is just scraping a scene url
				studioId := scene.Get("studio.slug").String()
				// see if we can find the site record, there may not be
				commonDb, _ := models.GetCommonDB()
				var site models.Site
				commonDb.Where(&models.Site{ID: sc.SiteID}).First(&site)

				site.GetIfExist(studioId)
				if site.Name != "" {
					log.Info("no vrporn scraper id using database %s", site.Name)
					// the user has setup a custom site, use the name they specified
					sc.Studio = site.Name
				} else {
					// the user has not setup a custom site, use the name from the api
					sc.Studio = scene.Get("studio.name").String()
					log.Info("no vrporn scraper id using api %s", sc.Studio)
				}
			}

			sc.Title = scene.Get("name").String()
			sc.Covers = append(sc.Covers, scene.Get("previewImage.path").String())
			sc.Synopsis = scene.Get("description").String()

			// Skipping some very generic and useless tags
			skiptags := map[string]bool{
				"3D":     true,
				"60 FPS": true,
				"HD":     true,
			}

			tagList := scene.Get("categories")
			tagList.ForEach(func(_, tag gjson.Result) bool {
				tagname := tag.Get("name").String()
				if !skiptags[tagname] {
					sc.Tags = append(sc.Tags, tagname)
				}
				return true
			})

			// Cast
			cast := scene.Get("models")
			sc.ActorDetails = make(map[string]models.ActorDetails)
			cast.ForEach(func(_, model gjson.Result) bool {
				model_gender := model.Get("gender").String()
				if model_gender != "male" {
					modelName := model.Get("name").String()
					modelSlug := model.Get("slug").String()
					sc.Cast = append(sc.Cast, modelName)
					modelImage := model.Get("image")
					actorApiUrl := "https://vrporn.com/proxy/api/content/v1/models/"
					if modelImage.Exists() {
						sc.ActorDetails[modelName] = models.ActorDetails{Source: "vrporn scrape", ImageUrl: modelImage.Get("path").String(), ProfileUrl: actorApiUrl + modelSlug}
					} else {
						sc.ActorDetails[modelName] = models.ActorDetails{Source: "vrporn scrape", ProfileUrl: actorApiUrl + modelSlug}
					}
				}
				return true
			})

			sc.Released = time.Unix(scene.Get("publishedAt").Int(), 0).Format("2006-01-02")
			sc.Duration = int(scene.Get("paidTime").Int() / 60)

			// trailer details
			sc.TrailerType = "vrporn"
			params := models.TrailerScrape{SceneUrl: "https://vrporn.com/proxy/api/content/v1/post/" + slug}
			strParams, _ := json.Marshal(params)
			sc.TrailerSrc = string(strParams)

			// gallery
			r, _ := resty.New().R().Get("https://vrporn.com/proxy/api/content/v1/videos/" + sc.SiteID + "/gallery")
			galleryJson := r.String()
			images := gjson.Get(galleryJson, "data")
			images.ForEach(func(_, image gjson.Result) bool {
				sc.Gallery = append(sc.Gallery, image.Get("path").String())
				return true
			})

			// Filenames need to build
			//see https://vrporn.com/sweet-teen-luci-shows-her-young-body-in-bleu-jeans/ as example of less resolutions
			out <- sc
		}

		// check if a single scene was returned or an array
		item := jsonData.Get("data.item")
		if item.Exists() {
			// single scene
			processScene(item)
		} else {
			// page list
			itemList := jsonData.Get("data.items")
			itemList.ForEach(func(_, scene gjson.Result) bool {
				slug := scene.Get("slug").String()
				if !funk.ContainsString(knownScenes, "https://vrporn.com/"+slug+"/") {
					apiCollector.Visit("https://vrporn.com/proxy/api/content/v1/post/" + slug)
				}
				return true
			})
			pages := jsonData.Get("data.pages").Int()
			if page < int(pages) && !limitScraping {
				page++
				slug := strings.TrimSuffix(strings.ReplaceAll(siteURL, "https://vrporn.com/studio/", ""), "/")
				url := "https://vrporn.com/proxy/api/content/v1/videos/studio/" + slug + "?page=" + strconv.Itoa(page) + "&limit=32&sort=new"
				WaitBeforeVisit("vrporn.com", apiCollector.Visit, url)
			}
		}
	})

	if singleSceneURL != "" {
		url := "https://vrporn.com/proxy/api/content/v1/post/" + path.Base(singleSceneURL)
		apiCollector.Visit(url)
	} else {
		slug := strings.TrimSuffix(strings.ReplaceAll(siteURL, "https://vrporn.com/studio/", ""), "/")
		//url:="https://vrporn.com/proxy/api/content/v1/videos/studio/"+slug+"?page=1&limit=32&sort=new&is-toys=true&is-ar=true"
		url := "https://vrporn.com/proxy/api/content/v1/videos/studio/" + slug + "?page=" + strconv.Itoa(page) + "&limit=32&sort=new"
		WaitBeforeVisit("vrporn.com", apiCollector.Visit, url)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addVRPornScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string, masterSiteId string) {
	suffixedName := name
	siteNameSuffix := name
	if custom {
		suffixedName += " (Custom VRPorn)"
		siteNameSuffix += " (VRPorn)"
	} else {
		suffixedName += " (VRPorn)"
	}
	if avatarURL == "" {
		avatarURL = "https://vrporn.com/favicon.ico"
	}

	if masterSiteId == "" {
		registerScraper(id, suffixedName, avatarURL, "vrporn.com", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return VRPorn(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, "")
		})
	} else {
		registerAlternateScraper(id, suffixedName, avatarURL, "vrporn.com", masterSiteId, func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return VRPorn(wg, updateSite, knownScenes, out, singleSceneURL, id, siteNameSuffix, company, siteURL, singeScrapeAdditionalInfo, limitScraping, masterSiteId)
		})
	}
}

func init() {
	registerScraper("vrporn-single_scene", "VRPorn - Other Studios", "", "vrporn.com", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return VRPorn(wg, updateSite, knownScenes, out, singleSceneURL, "", "", "", "", singeScrapeAdditionalInfo, limitScraping, "")
	})

	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.VrpornScrapers {
		addVRPornScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, false, scraper.URL, scraper.MasterSiteId)
	}
	for _, scraper := range scrapers.CustomScrapers.VrpornScrapers {
		addVRPornScraper(scraper.ID, scraper.Name, scraper.Company, scraper.AvatarUrl, true, scraper.URL, scraper.MasterSiteId)
	}
}

func VRPornTrailer(trailerConfig string) models.VideoSourceResponse {
	var videolist models.VideoSourceResponse
	var params models.TrailerScrape
	json.Unmarshal([]byte(trailerConfig), &params)

	// setup a request and get cookies/headers
	client := &http.Client{}
	req, err := http.NewRequest("GET", params.SceneUrl, nil)
	if err != nil {
		log.Infof("Error getting trailer info for %s, error %s", params.SceneUrl, err)
		return videolist
	}

	if params.KVHttpConfig == "" {
		params.KVHttpConfig = GetCoreDomain(params.SceneUrl) + "-trailers"
	}
	SetupHtmlRequest(params.KVHttpConfig, req)
	resp, err := client.Do(req)
	if err != nil {
		log.Infof("Error getting trailer info for %s, error %s", params.SceneUrl, err)
		return videolist
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	sceneJson := string(body)
	statusMsg := gjson.Get(sceneJson, "status.message").String()
	if statusMsg == "Ok" {
		sources := gjson.Get(sceneJson, "data.item.sources")
		sources.ForEach(func(sourceName, sourceValue gjson.Result) bool {
			group := sourceName.String()
			sourceValue.ForEach(func(videoName, video gjson.Result) bool {
				videolUrl := video.Get("path").String()
				videolist.VideoSources = append(videolist.VideoSources, models.VideoSource{URL: videolUrl, Quality: group + " - " + videoName.String()})
				return true // keep iterating
			})
			return true // keep iterating
		})
	}
	return videolist
}
