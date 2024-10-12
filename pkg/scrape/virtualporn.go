package scrape

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
)

func Project1ServiceAPI(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, siteData *siteMetaData, limitScraping bool) error {

	// this scraper is non-standard in that it gathers info via an api rather than scraping html pages
	defer wg.Done()

	logScrapeStart(siteData.scraperID, siteData.siteID)
	nextApiUrl := ""
	siteCollector := createCollector(siteData.baseURL)
	apiCollector := createCollector("site-api.project1service.com")
	offset := 0
	apiCollector.OnResponse(func(r *colly.Response) {
		sceneListJson := gjson.ParseBytes(r.Body)

		processScene := func(scene gjson.Result) {
			sc := models.ScrapedScene{}
			sc.ScraperID = siteData.scraperID
			sc.SceneType = "VR"
			sc.Studio = siteData.studio
			sc.Site = siteData.siteID
			id := strconv.Itoa(int(scene.Get("id").Int()))
			sc.SceneID = slugify.Slugify(sc.ScraperID) + "-" + id

			sc.Title = scene.Get("title").String()
			sc.HomepageURL = siteData.absoluteURL + `video/` + id + "/" + slugify.Slugify(strings.ReplaceAll(sc.Title, "'", ""))
			sc.MembersUrl = siteData.membersURL + id + "/" + slugify.Slugify(strings.ReplaceAll(sc.Title, "'", ""))

			sc.Synopsis = scene.Get("description").String()
			dateParts := strings.Split(scene.Get("dateReleased").String(), "T")
			sc.Released = dateParts[0]

			scene.Get("images.poster").ForEach(func(key, imgGroup gjson.Result) bool {
				if key.String() == "0" {
					imgurl := imgGroup.Get("xl.urls.webp").String()
					if imgurl != "" {
						sc.Covers = append(sc.Covers, imgurl)
					}

				} else {
					imgurl := imgGroup.Get("xl.urls.webp").String()
					if imgurl != "" {
						if len(sc.Covers) == 0 {
							sc.Covers = append(sc.Covers, imgurl)
						} else {
							sc.Gallery = append(sc.Gallery, imgurl)
						}
					}
				}
				return true
			})

			// Cast
			sc.ActorDetails = make(map[string]models.ActorDetails)
			scene.Get("actors").ForEach(func(key, actor gjson.Result) bool {
				name := actor.Get("name").String()
				if actor.Get("gender").String() == "female" {
					sc.Cast = append(sc.Cast, name)
				}
				sc.ActorDetails[actor.Get("name").String()] = models.ActorDetails{Source: scraperID + " scrape", ProfileUrl: siteData.modelURL + strconv.Itoa(int(actor.Get("id").Int())) + "/" + slugify.Slugify(name)}
				return true
			})

			// Tags
			scene.Get("tags").ForEach(func(key, tag gjson.Result) bool {
				if tag.Get("isVisible").Bool() {
					sc.Tags = append(sc.Tags, tag.Get("name").String())
				}
				return true
			})

			// trailer & filename details
			sc.TrailerType = "urls"
			var trailers []models.VideoSource
			scene.Get("children").ForEach(func(key, child gjson.Result) bool {
				child.Get("videos.full.files").ForEach(func(key, file gjson.Result) bool {
					quality := file.Get("format").String()
					url := file.Get("urls.view").String()
					filename := file.Get("urls.download").String()
					if url != "" {
						trailers = append(trailers, models.VideoSource{URL: url, Quality: quality})
					}
					pos := strings.Index(filename, "?filename=")
					if pos != -1 {
						sc.Filenames = append(sc.Filenames, filename[pos+10:])
					}
					return true
				})
				return true
			})
			trailerJson, _ := json.Marshal(models.VideoSourceResponse{VideoSources: trailers})
			sc.TrailerSrc = string(trailerJson)

			out <- sc

		}
		total := int(sceneListJson.Get("meta.total").Int())
		scenes := sceneListJson.Get("result")
		if strings.Contains(r.Request.URL.RawQuery, "offset=") {
			scenes.ForEach(func(key, scene gjson.Result) bool {
				// For some reason, the API will occasionally return results belonging to other studios filter them out
				if scene.Get("brand").String() == strings.ToLower(siteData.studio) {
					// check if we have the scene already
					matches := funk.Filter(knownScenes, func(s string) bool {
						return strings.Contains(s, scene.Get("id").String())
					})
					if funk.IsEmpty(matches) {
						processScene(scene)
					}
				}
				return true
			})
		} else {
			processScene(scenes)
		}

		offset += 24
		if offset < total {
			if !limitScraping {
				apiCollector.Visit("https://site-api.project1service.com/v2/releases?type=scene&limit=24&offset=" + strconv.Itoa(offset))
			}
		}
	})

	siteCollector.OnHTML(`script`, func(e *colly.HTMLElement) {
		// only interested in a script containg window\.__JUAN\.rawInstance
		re := regexp.MustCompile(`window\.__JUAN\.rawInstance = (\{.*?\});`)
		matches := re.FindStringSubmatch(e.Text)
		if len(matches) > 1 {
			instanceJson := gjson.ParseBytes([]byte(matches[1]))
			token := instanceJson.Get("jwt").String()
			// set up api requests to use the token in the Instance Header
			apiCollector.OnRequest(func(r *colly.Request) {
				r.Headers.Set("Instance", token)
				r.Headers.Set("Referer", siteData.absoluteURL)
				r.Headers.Set("Origin", siteData.absoluteURL)
			})
			apiCollector.Visit(nextApiUrl)
		}
	})
	if singleSceneURL != "" {
		ctx := colly.NewContext()
		ctx.Put("dur", "")
		ctx.Put("date", "")
		urlParts := strings.Split(singleSceneURL, "/")
		id := urlParts[len(urlParts)-2]
		offset = 9999 // do read more pages, we only need 1
		nextApiUrl = "https://site-api.project1service.com/v2/releases/" + id
		siteCollector.Visit(siteData.absoluteURL + `videos`)

	} else {
		// call virtualporn.com, this is just to get the instance token to use the api for this session
		nextApiUrl = "https://site-api.project1service.com/v2/releases?type=scene&limit=24&offset=" + strconv.Itoa(offset)
		siteCollector.Visit(siteData.absoluteURL + `videos`)
	}

	if updateSite {
		updateSiteLastUpdate(siteData.scraperID)
	}
	logScrapeFinished(siteData.scraperID, siteData.siteID)
	return nil
}

type siteMetaData struct {
	scraperID   string
	siteID      string
	modelURL    string
	absoluteURL string
	baseURL     string
	membersURL  string
	studio      string
}

func VirtualPorn(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	bvrMetaData := siteMetaData{
		scraperID:   "bvr",
		siteID:      "VirtualPorn",
		modelURL:    "https://virtualporn.com/model/",
		absoluteURL: "https://virtualporn.com/",
		baseURL:     "virtualporn.com",
		membersURL:  `https://site-ma.virtualporn.com/`,
		studio:      "BangBros",
	}
	return Project1ServiceAPI(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, &bvrMetaData, limitScraping)
}

func BrazzersVR(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	zzvrMetaData := siteMetaData{
		scraperID:   "zzvr",
		siteID:      "BrazzersVR",
		modelURL:    "https://www.brazzersvr.com/pornstar/",
		absoluteURL: "https://www.brazzersvr.com/",
		baseURL:     "www.brazzersvr.com",
		membersURL:  `https://site-ma.brazzersvr.com/`,
		studio:      "Brazzers",
	}
	return Project1ServiceAPI(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, &zzvrMetaData, limitScraping)
}

func init() {
	registerScraper("bvr", "VirtualPorn", "https://images.cn77nd.com/members/bangbros/favicon/apple-icon-60x60.png", "virtualporn.com", VirtualPorn)
	registerScraper("zzvr", "BrazzersVR", "https://images-assets-ht.project1content.com/BrazzersVR/Common/Favicon/63e2a8fdbdbe16.78976344.jpg", "brazzersvr.com", BrazzersVR)
}

// one off conversion routine called by migrations.go
func UpdateVirtualPornIds() {
	collector := createCollector("virtualporn.com")
	apiCollector := createCollector("site-api.project1service.com")
	offset := 0
	sceneCnt := 0

	collector.OnHTML(`script`, func(e *colly.HTMLElement) {
		// only interested in a script containg window\.__JUAN\.rawInstance
		re := regexp.MustCompile(`window\.__JUAN\.rawInstance = (\{.*?\});`)
		matches := re.FindStringSubmatch(e.Text)
		if len(matches) > 1 {
			instanceJson := gjson.ParseBytes([]byte(matches[1]))
			token := instanceJson.Get("jwt").String()
			// set up api requests to use the token in the Instance Header
			apiCollector.OnRequest(func(r *colly.Request) {
				r.Headers.Set("Instance", token)
			})
			apiCollector.Visit("https://site-api.project1service.com/v2/releases?type=scene&limit=100&offset=" + strconv.Itoa(offset))
		}
	})

	apiCollector.OnResponse(func(r *colly.Response) {
		db, _ := models.GetDB()
		defer db.Close()

		sceneListJson := gjson.ParseBytes(r.Body)
		sceneCnt = int(sceneListJson.Get("meta.total").Int())
		scenes := sceneListJson.Get("result")
		scenes.ForEach(func(key, apiScene gjson.Result) bool {
			id := strconv.Itoa(int(apiScene.Get("id").Int()))
			title := apiScene.Get("title").String()
			dateParts := strings.Split(apiScene.Get("dateReleased").String(), "T")
			releasedDate := dateParts[0]
			var scene models.Scene
			scene.GetIfExist("bvr-" + id)
			if scene.ID > 0 {
				// get the next record, this one already matches the new id
				return true
			}
			db.Where("scraper_id = ? and release_date_text = ?", "bvr", releasedDate).Find(&scene)
			if scene.ID > 0 {
				oldSceneId := scene.SceneID
				log.Infof("Updating SceneId %s to %s ", oldSceneId, "bvr-"+id)
				scene.LegacySceneID = scene.SceneID
				scene.SceneID = "bvr-" + id
				scene.SceneURL = "https://virtualporn.com/video/" + id + "/" + slugify.Slugify(strings.ReplaceAll(title, "'", ""))
				scene.MemberURL = "https://site-ma.virtualporn.com/scene/" + id + "/" + slugify.Slugify(strings.ReplaceAll(title, "'", ""))

				scene.Save()
				result := db.Model(&models.Action{}).Where("scene_id = ?", oldSceneId).Update("scene_id", scene.SceneID)
				if result.Error != nil {
					log.Infof("Converting Actions for VirtualPorn Scene %s to %s failed, %s", oldSceneId, scene.SceneID, result.Error)
				}
				result = db.Model(&models.ExternalReferenceLink{}).Where("internal_table = 'scenes' and internal_name_id = ?", oldSceneId).Update("internal_name_id", scene.SceneID)
				if result.Error != nil {
					log.Infof("Converting External Reference Links for VirtualPorn Scene %s to %s failed, %s", oldSceneId, scene.SceneID, result.Error)
				}
			}
			return true
		})
		offset += 100
		if offset < sceneCnt {
			apiCollector.Visit("https://site-api.project1service.com/v2/releases?type=scene&limit=24&offset=" + strconv.Itoa(offset))
		}
	})

	collector.Visit("https://virtualporn.com/videos")

	if sceneCnt == 0 {
		log.Info("Unable to access VirtualPorn scenes, existing scenes could not be upgraded with new Scene Ids")
	}

}
