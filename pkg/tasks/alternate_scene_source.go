package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func AddAlternateSceneSource(db *gorm.DB, scrapedScene models.ScrapedScene) {
	var extref models.ExternalReference
	source := "alternate scene " + scrapedScene.ScraperID
	extref.FindExternalId(source, scrapedScene.SceneID)
	extref.ExternalId = scrapedScene.SceneID
	extref.ExternalSource = source
	extref.ExternalURL = scrapedScene.HomepageURL

	var scene models.Scene
	scene.PopulateSceneFieldsFromExternal(db, scrapedScene)
	extref.ExternalDate = scene.ReleaseDate

	// strip out other actor columns, it makes the data too large and we only need the name
	var newCastList []models.Actor
	for _, actor := range scene.Cast {
		newCastList = append(newCastList, models.Actor{ID: actor.ID, Name: actor.Name})
	}
	scene.Cast = newCastList
	var data models.SceneAlternateSource
	data.MasterSiteId = scrapedScene.MasterSiteId
	data.Scene = scene
	data.MatchAchieved = -1
	jsonData, _ := json.Marshal(data)
	extref.ExternalData = string(jsonData)
	extref.UdfBool1 = scene.HumanScript
	extref.UdfBool2 = scene.AiScript
	extref.UdfDatetime1 = scene.ScriptPublished
	extref.Save()
}

func MatchAlternateSources() {
	tlog := log.WithField("task", "scrape")
	tlog.Info("Matching scenes from alternate sources")
	commonDb, _ := models.GetCommonDB()

	var unmatchedScenes []models.ExternalReference

	commonDb.Joins("Left JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source like 'alternate scene %' and erl.external_reference_id is NULL").
		Find(&unmatchedScenes)

	// check for scenes that should be relinked based on the reprocess_links param
	var sites []models.Site
	commonDb.Where("matching_params<> '' and JSON_EXTRACT(matching_params, '$.reprocess_links')>0").Find(&sites)
	for _, site := range sites {
		var reprocessList []models.ExternalReference
		var matchParams models.AltSrcMatchParams
		matchParams.UnmarshalParams(site.MatchingParams)
		reprocessDate := time.Now().AddDate(0, 0, matchParams.ReprocessLinks*-1)
		// ignore match_type 99999, they are manually match and shouldn't change
		commonDb.Preload("XbvrLinks").
			Joins("Join external_reference_links on external_reference_links.external_reference_id = external_references.id").
			Where("external_references.external_source = ? and external_references.external_date > ? and external_reference_links.match_type not in (-1, 99999)", "alternate scene "+site.ID, reprocessDate).Find(&reprocessList)
		unmatchedScenes = append(unmatchedScenes, reprocessList...)

	}
	lastProgressUpdate := time.Now()
	for cnt, altsource := range unmatchedScenes {
		if time.Since(lastProgressUpdate) > time.Duration(config.Config.Advanced.ProgressTimeInterval)*time.Second {
			tlog.Infof("Matching alternate scene sources %v of %v", cnt, len(unmatchedScenes))
			lastProgressUpdate = time.Now()
		}
		var unmatchedSceneData models.SceneAlternateSource
		var possiblematchs []models.Scene
		var site models.Site
		var matchParams models.AltSrcMatchParams
		json.Unmarshal([]byte(altsource.ExternalData), &unmatchedSceneData)
		site.GetIfExist(strings.TrimPrefix(altsource.ExternalSource, "alternate scene "))
		err := matchParams.UnmarshalParams(site.MatchingParams)
		if err != nil {
			tlog.Infof("Cannot read Matching parameters for %s %s", site.Name, err)
			continue
		}
		if matchParams.DelayLinking > 0 {
			skipDate := altsource.ExternalDate.AddDate(0, 0, matchParams.DelayLinking)
			if skipDate.After(time.Now()) {
				continue
			}
		}

		if !matchParams.IgnoreReleasedBefore.IsZero() {
			if unmatchedSceneData.Scene.ReleaseDate.Before(matchParams.IgnoreReleasedBefore) {
				continue
			}
		} else {
			if !config.Config.Advanced.IgnoreReleasedBefore.IsZero() {
				if unmatchedSceneData.Scene.ReleaseDate.Before(config.Config.Advanced.IgnoreReleasedBefore) {
					continue
				}
			}
		}

		tmpTitle := strings.ReplaceAll(strings.ReplaceAll(unmatchedSceneData.Scene.Title, " and ", "&"), "+", "&")
		for _, char := range `'- :!?"/.,@()–` {
			tmpTitle = strings.ReplaceAll(tmpTitle, string(char), "")
		}
		commonDb.Preload("Cast").Where(`replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(title,'''',''),'-',''),' ',''),':',''),'!',''),'?',''),'"',''),'/',''),'.',''),',',''),' and ','&'),'@',''),')',''),'(',''), '+','&'),'–','') like ? and scraper_id = ?`,
			tmpTitle, unmatchedSceneData.MasterSiteId).Find(&possiblematchs)

		if len(possiblematchs) == 1 {
			found := false
			// check not already linked, if we update unnessarily, it will mess up the sort by "Released on Alternate Sites" sort option
			for _, link := range altsource.XbvrLinks {
				if link.InternalDbId == possiblematchs[0].ID {
					found = true
					break
				}
			}
			if !found {
				UpdateLinks(commonDb, altsource.ID, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: possiblematchs[0].ID, InternalNameId: possiblematchs[0].SceneID,
					ExternalReferenceID: altsource.ID, ExternalSource: altsource.ExternalSource, ExternalId: altsource.ExternalId, MatchType: 10000, UdfDatetime1: time.Now()})
			}
		} else {
			// build the search query based on the sites matching params
			ignoreChars := `+-=&|><!(){}[]^\"~*?:\\/_–`
			title := unmatchedSceneData.Scene.Title
			for _, char := range ignoreChars {
				title = strings.ReplaceAll(title, string(char), " ")
			}

			desc := unmatchedSceneData.Scene.Synopsis
			for _, char := range ignoreChars {
				desc = strings.ReplaceAll(desc, string(char), " ")
			}

			// get the name from the master site, thats what's in the search indexes
			sitename := ""
			var masterSite models.Site
			masterSite.GetIfExist(unmatchedSceneData.MasterSiteId)
			sitename = masterSite.Name
			if strings.Index(masterSite.Name, `(`) > 0 {
				sitename = masterSite.Name[:strings.Index(masterSite.Name, `(`)-1]
			}

			q := fmt.Sprintf(`+site:"%s" title:"%s"^%v`, sitename, title, matchParams.BoostTitleExact)
			for _, word := range strings.Fields(title) {
				q = fmt.Sprintf(`%s title:%s^%v`, q, word, matchParams.BoostTitleAnyWords)
			}

			if matchParams.CastMatchType == "should" {
				for _, actor := range unmatchedSceneData.Scene.Cast {
					q = fmt.Sprintf(`%s cast:"%s"^%v`, q, actor.Name, matchParams.BoostCast)
					// split to indivual words as well, eg handles cases like "someone baberoticvr" or one site only using a first name
					// but lower the boosting a bit
					words := strings.Fields(actor.Name)
					for _, word := range words {
						q = fmt.Sprintf(`%s cast:%s^%0.2f`, q, word, matchParams.BoostCast*0.75)
					}
				}
			}

			if matchParams.DurationMatchType != "do not" {
				// note: no boosting for duration, doesn't appear to work except on ranges, where it must be within the range anyway
				if matchParams.DurationMatchType == "must" {
					matchParams.DurationRangeLess = 0
					matchParams.DurationRangeMore = 0
				}
				if unmatchedSceneData.Scene.Duration >= matchParams.DurationMin {
					if matchParams.DurationRangeLess+matchParams.DurationRangeMore > 1 {
						q = fmt.Sprintf("%s duration: %v duration:>=%v duration:<=%v", q, unmatchedSceneData.Scene.Duration, unmatchedSceneData.Scene.Duration-matchParams.DurationRangeLess, unmatchedSceneData.Scene.Duration+matchParams.DurationRangeMore)
					} else {
						q = fmt.Sprintf("%s duration: %v", q, unmatchedSceneData.Scene.Duration)
					}
				}
			}

			if matchParams.ReleasedMatchType != "do not" {
				if matchParams.ReleasedMatchType == "must" {
					matchParams.ReleasedPrior = 0
					matchParams.ReleasedAfter = 0
				}
				if matchParams.IgnoreReleasedBefore.IsZero() || unmatchedSceneData.Scene.ReleaseDate.After(matchParams.IgnoreReleasedBefore.AddDate(0, 0, -1)) {
					prefix := ""
					if matchParams.ReleasedMatchType == "must" {
						prefix = "+"
					}
					q = fmt.Sprintf(`%s %sreleased:>="%s"^%v %sreleased:<="%s"^%v`, q,
						prefix, unmatchedSceneData.Scene.ReleaseDate.AddDate(0, 0, matchParams.ReleasedPrior*-1).Format("2006-01-02"), matchParams.BoostReleased,
						prefix, unmatchedSceneData.Scene.ReleaseDate.AddDate(0, 0, matchParams.ReleasedAfter).Format("2006-01-02"), matchParams.BoostReleased)
				}
			}

			if matchParams.DescriptionMatchType == "should" { // "must" is not an option for description
				words := strings.Fields(desc)
				for _, word := range words {
					if strings.TrimSpace(word) != "" {
						q = fmt.Sprintf("%s description: %v^%v", q, strings.TrimSpace(word), matchParams.BoostDescription)
					}
				}
			}

			query := bleve.NewQueryStringQuery(q)

			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"title", "cast", "site", "description", "released"}
			searchRequest.IncludeLocations = true
			searchRequest.From = 0
			searchRequest.Size = 25
			searchRequest.SortBy([]string{"-_score"})

			searchResults, err := AltSourceSearch(searchRequest)
			if err != nil {
				log.Error(err)
				log.Error(q)
				continue
			}

			var extdata models.SceneAlternateSource
			// remove saving the query later, keep for debug purposes
			json.Unmarshal([]byte(altsource.ExternalData), &extdata)
			extdata.Query = q
			newjson, _ := json.Marshal(extdata)
			altsource.ExternalData = string(newjson)
			tmpAltSource := altsource
			tmpAltSource.XbvrLinks = nil
			// save the updated query used, but don't update the links
			tmpAltSource.Save()
			if len(searchResults.Hits) > 0 {
				var scene models.Scene
				scene.GetIfExist(searchResults.Hits[0].ID)
				if scene.ID > 0 {
					// check not already linked, if we update unnessarily, it will mess up the sort by "Released on Alternate Sites" sort option
					found := false
					for _, link := range altsource.XbvrLinks {
						if link.InternalDbId == scene.ID {
							found = true
							break
						}
					}
					if !found {
						UpdateLinks(commonDb, altsource.ID, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: scene.ID, InternalNameId: scene.SceneID,
							ExternalReferenceID: altsource.ID, ExternalSource: altsource.ExternalSource, ExternalId: altsource.ExternalId, MatchType: int(searchResults.Hits[0].Score), UdfDatetime1: time.Now()})
					}
				}
			}
		}
	}
	tlog.Info("Completed Matching scenes from alternate sources")
}
func AltSourceSearch(searchRequest *bleve.SearchRequest) (*bleve.SearchResult, error) {
	// open and close the search for each search, this stops the search function from locking users out of searching
	idx, err := NewIndex("scenes")
	if err != nil {
		return nil, err
	}
	defer idx.Bleve.Close()
	return idx.Bleve.Search(searchRequest)
}
func UpdateLinks(db *gorm.DB, externalreference_id uint, newLink models.ExternalReferenceLink) {
	var extref models.ExternalReference
	extref.GetIfExist(externalreference_id)
	exists := false

	db.Where("external_source = ? and external_reference_id = ? and internal_db_id <> ?").Find(&extref)
	for _, link := range extref.XbvrLinks {
		if link.InternalDbId == newLink.InternalDbId {
			exists = true
		} else {
			link.Delete()
		}
	}
	if !exists {
		newLink.Save()
	}

}
