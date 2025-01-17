package scrape

import (
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
)

func StashStudio(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, scraper string, name string, limitScraping bool, stashGuid string, masterSiteId string) error {
	defer wg.Done()
	commonDb, _ := models.GetCommonDB()
	stashGuid = strings.TrimPrefix(stashGuid, "https://stashdb.org/studios/")

	scraperID := scraper
	siteID := name
	logScrapeStart(scraperID, siteID)

	if singleSceneURL != "" {
		stashGuid := strings.TrimPrefix(strings.ToLower(singleSceneURL), "https://stashdb.org/scenes/")
		stashScene := GetSceneFromStash(stashGuid)
		if stashScene.ID != "" {
			sc := processScrapedScene(stashScene, "", commonDb)
			out <- sc
		}
	} else {
		scenes := getStashdbScenes(stashGuid, "", "", limitScraping)
		for _, stashScene := range scenes.Data.QueryScenes.Scenes {
			if !funk.ContainsString(knownScenes, "https://stashdb.org/scenes/"+stashScene.ID) {
				sc := processScrapedScene(stashScene, masterSiteId, commonDb)
				out <- sc
			}
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}
func processScrapedScene(stashScene models.StashScene, masterSiteId string, commonDb *gorm.DB) models.ScrapedScene {
	var scene models.Scene
	scene.GetIfExist("stash-" + stashScene.ID)

	sc := models.ScrapedScene{}
	sc.MasterSiteId = masterSiteId
	sc.ScraperID = slugify.Slugify(stashScene.Studio.Name) + "-stashdb"
	sc.SceneType = "2D"
	sc.Studio = stashScene.Studio.Name
	if stashScene.Studio.Parent.Name != "" {
		sc.Studio = stashScene.Studio.Parent.Name
	}
	sc.Site = stashScene.Studio.Name + " (Stash)"
	sc.HomepageURL = "https://stashdb.org/scenes/" + stashScene.ID
	sc.SceneID = "stash-" + stashScene.ID

	for _, urldata := range stashScene.URLs {
		if urldata.Type == "STUDIO" {
			sc.MembersUrl = urldata.URL
			continue
		}
	}
	sc.Released = stashScene.Date
	sc.Title = stashScene.Title
	sc.Synopsis = stashScene.Details
	if len(stashScene.Images) > 0 {
		sc.Covers = append(sc.Covers, stashScene.Images[0].URL)
	}

	for _, tag := range stashScene.Tags {
		sc.Tags = append(sc.Tags, tag.Name)
		if tag.Name == "Virtual Reailty" || tag.Name == "VR" {
			sc.SceneType = "VR"
		}
	}

	// Cast
	sc.ActorDetails = make(map[string]models.ActorDetails)
	for _, model := range stashScene.Performers {
		modelName := model.Performer.Name
		if model.Performer.Disambiguation != "" {
			modelName = model.Performer.Name + "(" + model.Performer.Disambiguation + ")"
		}
		sc.Cast = append(sc.Cast, modelName)

		tmpActor := models.Actor{}
		commonDb.Where(&models.Actor{Name: strings.Replace(modelName, ".", "", -1)}).First(&tmpActor)
		if tmpActor.ID == 0 {
			commonDb.Where(&models.Actor{Name: strings.Replace(modelName, ".", "", -1)}).FirstOrCreate(&tmpActor)
			stashPerformer := GetStashPerformer(model.Performer.ID)
			externalreference.UpdateXbvrActor(stashPerformer.Data.Performer, tmpActor.ID)
		}
	}

	sc.Duration = stashScene.Duration / 60
	return sc
}

func init() {
	addStashScraper("single_scene", "Stashdb - Other", "", "", "")
	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.StashDbScrapers {
		addStashScraper(slugify.Slugify(scraper.Name), scraper.Name, scraper.AvatarUrl, scraper.URL, scraper.MasterSiteId)
	}
	for _, scraper := range scrapers.CustomScrapers.StashDbScrapers {
		addStashScraper(slugify.Slugify(scraper.Name), scraper.Name, scraper.AvatarUrl, scraper.URL, scraper.MasterSiteId)
	}
}
func addStashScraper(id string, name string, avatarURL string, stashGuid string, masterSiteId string) {
	if masterSiteId == "" {
		registerScraper(id+"-stashdb", name+" (Stashdb)", avatarURL, "stashdb.org", func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return StashStudio(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping, stashGuid, masterSiteId)
		})
	} else {
		registerAlternateScraper(id+"-stashdb", name+" (Stashdb)", avatarURL, "stashdb.org", masterSiteId, func(wg *models.ScrapeWG, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return StashStudio(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping, stashGuid, masterSiteId)
		})
	}
}

func getStashdbScenes(studioId string, parentId string, tagId string, limitScraping bool) QueryScenesResult {
	const count = 25
	page := 1
	var sceneList QueryScenesResult
	var nextList QueryScenesResult
	var variables string
	if parentId != "" {
		variables = getParentSceneQueryVariable(parentId, tagId, page, count)
	} else {
		variables = getStudioSceneQueryVariable(studioId, page, count)
	}
	sceneList = GetScenePage(variables)
	nextList = sceneList
	for limitScraping == false &&
		len(nextList.Data.QueryScenes.Scenes) > 0 &&
		len(sceneList.Data.QueryScenes.Scenes) < sceneList.Data.QueryScenes.Count {
		page += 1
		if parentId != "" {
			variables = getParentSceneQueryVariable(parentId, tagId, page, count)
		} else {
			variables = getStudioSceneQueryVariable(studioId, page, count)
		}
		nextList = GetScenePage(variables)
		sceneList.Data.QueryScenes.Scenes = append(sceneList.Data.QueryScenes.Scenes, nextList.Data.QueryScenes.Scenes...)
	}
	return sceneList
}
