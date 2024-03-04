package externalreference

import (
	"encoding/json"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/xbapps/xbvr/pkg/models"
)

func UpdateAllPerformerData() {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Starting Updating Actor Details ")
	db, _ := models.GetDB()
	defer db.Close()

	var performers []models.ExternalReference

	db.Preload("XbvrLinks").
		Joins("JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source = 'stashdb performer'").
		Find(&performers)
		// join actors test image url/arr =''

	for _, performer := range performers {
		var data models.StashPerformer
		json.Unmarshal([]byte(performer.ExternalData), &data)

		if len(data.Images) > 0 {
			for _, actorLink := range performer.XbvrLinks {
				var actor models.Actor
				db.Where(models.Actor{ID: actorLink.InternalDbId}).Find(&actor)
				if actor.ImageUrl == "" || actor.ImageArr == "" {
					UpdateXbvrActor(data, actor.ID)

				}

			}
		}
	}
	tlog.Infof("Updating Actor Images Completed")
}

// this applies rules for matching xbvr scenes to stashdb, it then check if any matched scenes can be used to match actors
func ApplySceneRules() {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Starting Scene Rule Matching")

	matchOnSceneUrl()
	config := models.BuildActorScraperRules()

	for sitename, configSite := range config.StashSceneMatching {
		if len(configSite.Rules) > 0 {
			if configSite.StashId == "" {
				var ext models.ExternalReference
				ext.FindExternalId("stashdb studio", sitename)
				configSite.StashId = ext.ExternalId
			}
			matchSceneOnRules(sitename, config)
		}
	}

	checkMatchedScenes()
	tlog.Infof("Scene Rule Matching Completed")
}

// if an unmatched scene has a trailing number try to match on the  xbvr scene_id for that studio
func matchOnSceneUrl() {

	db, _ := models.GetDB()
	defer db.Close()

	var stashScenes []models.ExternalReference
	var unmatchedXbvrScenes []models.Scene

	db.Joins("Left JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source = ? and erl.internal_db_id is null", "stashdb scene").
		Find(&stashScenes)

	db.Joins("left join external_reference_links erl on erl.internal_db_id = scenes.id and external_source='stashdb scene'").
		Where("erl.id is null").
		Find(&unmatchedXbvrScenes)

	for _, stashScene := range stashScenes {
		var scene models.StashScene
		json.Unmarshal([]byte(stashScene.ExternalData), &scene)
		var xbvrId uint
		var xbvrSceneId string

		// see if we can link to an xbvr scene based on the urls
		for _, url := range scene.URLs {
			if url.Type == "STUDIO" {
				var xbvrScene models.Scene
				for _, scene := range unmatchedXbvrScenes {
					sceneurl := removeQueryFromURL(scene.SceneURL)
					tmpurl := removeQueryFromURL(url.URL)
					sceneurl = simplifyUrl(sceneurl)
					tmpurl = simplifyUrl(tmpurl)
					if strings.EqualFold(sceneurl, tmpurl) {
						xbvrScene = scene
					}
				}
				if xbvrScene.ID != 0 {
					xbvrId = xbvrScene.ID
					xbvrSceneId = xbvrScene.SceneID
				}
			}
		}
		if xbvrId != 0 {
			var xbrLink []models.ExternalReferenceLink
			xbrLink = append(xbrLink, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: xbvrId, InternalNameId: xbvrSceneId, ExternalSource: "stashdb scene", ExternalId: scene.ID, MatchType: 10})
			stashScene.XbvrLinks = xbrLink
			stashScene.AddUpdateWithId()
		}
	}
}
func removeQueryFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.RawQuery = ""
	lastSlashIndex := strings.LastIndex(parsedURL.Path, "/")
	if lastSlashIndex == -1 {
		// No forward slash found, return the original input
		return parsedURL.Path
	}
	cleanedURL := parsedURL.String()
	return cleanedURL
}
func simplifyUrl(url string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(url, "http://", ""), "https://", ""), "www.", ""), "/", ""), "-", ""), "_", "")
}

// if an unmatched scene has a trailing number try to match on the  xbvr scene_id for that studio
func matchSceneOnRules(sitename string, config models.ActorScraperConfig) {

	db, _ := models.GetDB()
	defer db.Close()

	if config.StashSceneMatching[sitename].StashId == "" {
		var ext models.ExternalReference
		ext.FindExternalId("stashdb studios", sitename)
		site := config.StashSceneMatching[sitename]
		site.StashId = ext.ExternalId
		config.StashSceneMatching[sitename] = site
	}

	log.Infof("Matching on rules for %s Stashdb Id: %s", sitename, config.StashSceneMatching[sitename].StashId)
	var stashScenes []models.ExternalReference
	stashId := config.StashSceneMatching[sitename].StashId
	if stashId == "" {
		return
	}

	var xbrScenes []models.Scene
	db.Preload("Cast").Where("scraper_id = ?", sitename).Find(&xbrScenes)

	db.Joins("Left JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Where("external_references.external_source = ? and erl.internal_db_id is null and external_data like ?", "stashdb scene", "%"+stashId+"%").
		Find(&stashScenes)

	for _, stashScene := range stashScenes {
		var data models.StashScene
		json.Unmarshal([]byte(stashScene.ExternalData), &data)
	urlLoop:
		for _, url := range data.URLs {
			if url.Type == "STUDIO" {
				for _, rule := range config.StashSceneMatching[sitename].Rules { // for each rule on this site
					re := regexp.MustCompile(rule.StashRule)
					match := re.FindStringSubmatch(url.URL)
					if match != nil {
						var extrefSite models.ExternalReference
						db.Where("external_source = ? and external_id = ?", "stashdb studio", data.Studio.ID).Find(&extrefSite)
						if extrefSite.ID != 0 {
							var xbvrScene models.Scene
							switch rule.XbvrField {
							case "scene_id":
								for _, scene := range xbrScenes {
									if strings.HasSuffix(scene.SceneID, match[rule.StashMatchResultPosition]) {
										xbvrScene = scene
										break
									}
								}
							case "scene_url":
								for _, scene := range xbrScenes {
									if strings.Contains(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(scene.SceneURL), "-", " "), "_", " "), strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(match[rule.StashMatchResultPosition]), "-", " "), "_", " ")) {
										xbvrScene = scene
										break
									}
								}
							default:
								log.Errorf("Unkown xbvr field %s", rule.XbvrField)
							}

							if xbvrScene.ID != 0 {
								xbvrLink := models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: xbvrScene.ID, InternalNameId: xbvrScene.SceneID,
									ExternalReferenceID: stashScene.ID, ExternalSource: stashScene.ExternalSource, ExternalId: stashScene.ExternalId, MatchType: 20}
								stashScene.XbvrLinks = append(stashScene.XbvrLinks, xbvrLink)
								stashScene.Save()
								matchPerformerName(data, xbvrScene, 20)
								break urlLoop
							}
						}
					}
				}
			}
		}
	}

}

// checks if scenes that have a match, can match the scenes performers
func checkMatchedScenes() {
	db, _ := models.GetDB()
	defer db.Close()
	var stashScenes []models.ExternalReference
	db.Joins("JOIN external_reference_links erl on erl.external_reference_id = external_references.id").
		Preload("XbvrLinks").
		Where("external_references.external_source = ?", "stashdb scene").
		Find(&stashScenes)

	for _, extref := range stashScenes {
		var scene models.StashScene
		err := json.Unmarshal([]byte(extref.ExternalData), &scene)
		if err != nil {
			log.Infof("checkMatchedScenes %s %s %s", err, scene.ID, scene.Title)
		}
		var xbvrScene models.Scene

		for _, link := range extref.XbvrLinks {
			db.Where("id = ?", link.InternalDbId).Preload("Cast").Find(&xbvrScene)
			if xbvrScene.ID != 0 {

				for _, performer := range scene.Performers {
					var ref models.ExternalReference
					db.Preload("XbvrLinks").Where(&models.ExternalReference{ExternalSource: "stashdb performer", ExternalId: performer.Performer.ID}).Find(&ref)
					if ref.ID == 0 {
						continue
					}
					var fullPerformer models.StashPerformer
					err := json.Unmarshal([]byte(ref.ExternalData), &fullPerformer)
					if err != nil {
						log.Infof("checkMatchedScenes %s %s %s", err, fullPerformer.ID, fullPerformer.Name)
					}

					// if len(ref.XbvrLinks) == 0 {
					for _, xbvrActor := range xbvrScene.Cast {
						if strings.EqualFold(strings.TrimSpace(xbvrActor.Name), strings.TrimSpace(performer.Performer.Name)) {
							// check if actor already matched
							exists := false
							for _, link := range ref.XbvrLinks {
								if link.InternalDbId == xbvrActor.ID {
									exists = true
								}
							}
							if !exists {
								xbrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: xbvrActor.ID, InternalNameId: xbvrActor.Name,
									ExternalReferenceID: ref.ID, ExternalSource: ref.ExternalSource, ExternalId: ref.ExternalId, MatchType: link.MatchType}
								ref.XbvrLinks = append(ref.XbvrLinks, xbrLink)
								ref.AddUpdateWithId()
								UpdateXbvrActor(fullPerformer, xbvrActor.ID)
							}
						}
					}
				}
			}
		}
	}
}

// updates an xbvr actor with data from a match stashdb actor
func UpdateXbvrActor(performer models.StashPerformer, xbvrActorID uint) {
	db, _ := models.GetDB()
	defer db.Close()

	changed := false
	actor := models.Actor{ID: xbvrActorID}
	err := db.Where(&actor).First(&actor).Error
	if err != nil {
		return
	}

	if len(performer.Images) > 0 {
		if actor.ImageUrl != performer.Images[0].URL && !actor.CheckForSetImage() {
			changed = true
			actor.ImageUrl = performer.Images[0].URL
		}
	}
	for _, alias := range performer.Aliases {
		changed = actor.AddToAliases(alias) || changed
	}
	if !strings.EqualFold(actor.Name, performer.Name) {
		changed = actor.AddToAliases(performer.Name) || changed
	}

	changed = CheckAndSetStringActorField(&actor.Gender, "gender", performer.Gender, actor.ID) || changed
	changed = CheckAndSetDateActorField(&actor.BirthDate, "birth_date", performer.BirthDate, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.Nationality, "nationality", performer.Country, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.Ethnicity, "ethnicity", performer.Ethnicity, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.Height, "height", performer.Height, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.EyeColor, "eye_color", performer.EyeColor, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.HairColor, "eye_color", performer.HairColor, actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.CupSize, "cup_size", performer.CupSize, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.BandSize, "band_size", int(math.Round(float64(performer.BandSize)*2.54)), actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.HipSize, "hip_size", int(math.Round(float64(performer.HipSize)*2.54)), actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.WaistSize, "waist_size", int(math.Round(float64(performer.WaistSize)*2.54)), actor.ID) || changed
	changed = CheckAndSetStringActorField(&actor.BreastType, "breast_type", performer.BreastType, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.StartYear, "start_year", performer.CareerStartYear, actor.ID) || changed
	changed = CheckAndSetIntActorField(&actor.EndYear, "end_year", performer.CareerEndYear, actor.ID) || changed

	for _, tattoo := range performer.Tattoos {
		tattooString := convertBodyModToString(tattoo)
		if !actor.CheckForUserDeletes("tattoos", tattooString) {
			changed = actor.AddToTattoos(tattooString) || changed
		}
	}
	for _, piercing := range performer.Piercings {
		piercingString := convertBodyModToString(piercing)
		if !actor.CheckForUserDeletes("piercings", piercingString) {
			changed = actor.AddToPiercings(piercingString) || changed
		}
	}
	for _, img := range performer.Images {
		if !actor.CheckForUserDeletes("image_arr", img.URL) {
			changed = actor.AddToImageArray(img.URL) || changed
		}
	}
	for _, url := range performer.URLs {
		if !actor.CheckForUserDeletes("urls", url.URL) {
			changed = actor.AddToActorUrlArray(models.ActorLink{Url: url.URL, Type: ""}) || changed
		}
	}
	if changed {
		actor.Save()
	}
}

func convertBodyModToString(bodyMod models.StashBodyModification) string {

	newMod := ""
	if bodyMod.Location != "" {
		newMod = bodyMod.Location
	}
	if bodyMod.Description != "" {
		if newMod != "" {
			newMod += " "
		}
		newMod += bodyMod.Description
	}
	return newMod
}

func matchPerformerName(scene models.StashScene, xbvrScene models.Scene, matchLevl int) {
	db, _ := models.GetDB()
	defer db.Close()

	for _, performer := range scene.Performers {
		var ref models.ExternalReference
		db.Preload("XbvrLinks").Where(&models.ExternalReference{ExternalSource: "stashdb performer", ExternalId: performer.Performer.ID}).Find(&ref)

		if ref.ID != 0 && len(ref.XbvrLinks) == 0 {
			for _, xbvrActor := range xbvrScene.Cast {
				if strings.EqualFold(strings.TrimSpace(xbvrActor.Name), strings.TrimSpace(performer.Performer.Name)) {
					xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: xbvrActor.ID, InternalNameId: xbvrActor.Name, MatchType: matchLevl,
						ExternalReferenceID: ref.ID, ExternalSource: ref.ExternalSource, ExternalId: ref.ExternalId}
					ref.XbvrLinks = append(ref.XbvrLinks, xbvrLink)
					ref.AddUpdateWithId()
					var data models.StashPerformer
					json.Unmarshal([]byte(ref.ExternalData), &data)
					UpdateXbvrActor(data, xbvrActor.ID)

					actor := models.Actor{ID: xbvrActor.ID}
					db.Where(&actor).First(&actor)
					if actor.ImageUrl == "" {
						var data models.StashPerformer
						json.Unmarshal([]byte(ref.ExternalData), &data)
						if len(data.Images) > 0 {
							actor.ImageUrl = data.Images[0].URL
							actor.Save()
						}
					}
				}
			}
		}
	}

}

// tries to match from stash to xbvr using the aka or aliases from stash
func MatchAkaPerformers() {
	tlog := log.WithField("task", "scrape")
	tlog.Info("Starting Match on Actor Aka/Aliases")
	db, _ := models.GetDB()
	defer db.Close()

	type AkaList struct {
		ActorId           string
		AkaName           string
		SceneInternalDbId int
		Aliases           string
	}
	var akaList []AkaList

	var sqlcmd string

	// find performers, that are unmatched, get their scenes, cross join with their aliases
	switch db.Dialect().GetName() {
	case "mysql":
		sqlcmd = `
		select trim('"' from json_extract(value, '$.Performer.id')) as actor_id, trim('"' from json_extract(value, '$.As')) as aka_name, erl_s.internal_db_id scene_internal_db_id, json_extract(value, '$.Performer.aliases') as aliases
		FROM external_references er_p
		left join external_reference_links erl_p on erl_p.external_reference_id = er_p.id
		JOIN external_references er_s on er_s.external_data like CONCAT('%', er_p.external_id, '%') 
		join external_reference_links erl_s on erl_s.external_reference_id = er_s.id
		JOIN JSON_TABLE(er_s.external_data , '$.performers[*]' COLUMNS(value JSON PATH '$' )) u
		where er_p.external_source ='stashdb performer' and erl_p.internal_db_id is null
		`
	case "sqlite3":
		sqlcmd = `
		select json_extract(value, '$.Performer.id') as actor_id, json_extract(value, '$.As') as aka_name, erl_s.internal_db_id scene_internal_db_id,  json_extract(value, '$.Performer.aliases') as aliases
		from external_references er_p  
		left join external_reference_links erl_p on erl_p.external_reference_id = er_p.id
		join external_references er_s on er_s.external_data like '%' || er_p.external_id || '%'
		join external_reference_links erl_s on erl_s.external_reference_id = er_s.id
		Cross Join json_each(json_extract(er_s.external_data, '$.performers')) j
		where er_p.external_source ='stashdb performer' and erl_p.internal_db_id is null
		`
	}
	db.Raw(sqlcmd).Scan(&akaList)

	for _, aka := range akaList {
		var scene models.Scene
		scene.GetIfExistByPK(uint(aka.SceneInternalDbId))
		for _, actor := range scene.Cast {
			var extref models.ExternalReference
			if strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(aka.AkaName)) {
				extref.FindExternalId("stashdb performer", aka.ActorId)
				if extref.ID != 0 && len(extref.XbvrLinks) == 0 {
					xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 30,
						ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
					extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
					extref.Save()
					var data models.StashPerformer
					json.Unmarshal([]byte(extref.ExternalData), &data)
					UpdateXbvrActor(data, actor.ID)
				}
			}
			if len(extref.XbvrLinks) == 0 {
				var aliases []string
				json.Unmarshal([]byte(aka.Aliases), &aliases)
				for _, alias := range aliases {
					if len(extref.XbvrLinks) == 0 && strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(alias)) {
						extref.FindExternalId("stashdb performer", aka.ActorId)
						if extref.ID != 0 && len(extref.XbvrLinks) == 0 {
							xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 30,
								ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
							extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
							extref.Save()
							var data models.StashPerformer
							json.Unmarshal([]byte(extref.ExternalData), &data)
							UpdateXbvrActor(data, actor.ID)

						}
					}
				}
			}
		}
	}
	ReverseMatch()
	LinkOnXbvrAkaGroups()
	// reapply edits in case manual change if match_cycle
	tlog.Info("Match on Actor Aka/Aliases completed")
}

// we match from an xbvr back to stash for cases where the Stash actor name or aka used is different to the xbvr actor name
// if the scene was matched, then we can check the stash actors aliases for a match
func ReverseMatch() {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Starting actor match from XBVR to Stashdb ")
	db, _ := models.GetDB()
	defer db.Close()
	var unmatchedActors []models.Actor
	var externalScenes []models.ExternalReference

	// get a list of unmatch xbvr actors
	db.Table("actors").Joins("LEFT JOIN external_reference_links erl on erl.internal_db_id =actors.id and erl.external_source ='stashdb performer'").Where("erl.internal_db_id is null").Find(&unmatchedActors)

	for _, actor := range unmatchedActors {
		// find scenes for the actor that have been matched
		db.Table("scene_cast").
			Joins("JOIN external_reference_links erl on erl.internal_db_id = scene_cast.scene_id and erl.external_source = 'stashdb scene'").
			Joins("JOIN external_references er on er.id =erl.external_reference_id").
			Select("er.*").
			Where("actor_id = ?", actor.ID).
			Find(&externalScenes)
	sceneLoop:
		for _, stashScene := range externalScenes {
			var stashSceneData models.StashScene
			json.Unmarshal([]byte(stashScene.ExternalData), &stashSceneData)
			for _, performance := range stashSceneData.Performers {
				if strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(performance.As)) {
					var extref models.ExternalReference
					extref.FindExternalId("stashdb performer", performance.Performer.ID)
					if extref.ID != 0 {
						xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 40,
							ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
						extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
						extref.Save()
						var data models.StashPerformer
						json.Unmarshal([]byte(extref.ExternalData), &data)
						UpdateXbvrActor(data, actor.ID)

					} else {
						log.Info("match no actor")
					}
					break sceneLoop
				}
				for _, alias := range performance.Performer.Aliases {
					if strings.EqualFold(strings.TrimSpace(actor.Name), strings.TrimSpace(alias)) {
						var extref models.ExternalReference
						extref.FindExternalId("stashdb performer", performance.Performer.ID)
						if extref.ID != 0 {
							xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 40,
								ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId}
							extref.XbvrLinks = append(extref.XbvrLinks, xbvrLink)
							extref.Save()
							var data models.StashPerformer
							json.Unmarshal([]byte(extref.ExternalData), &data)
							UpdateXbvrActor(data, actor.ID)
						} else {
							var data models.StashPerformer
							json.Unmarshal([]byte(extref.ExternalData), &data)
							UpdateXbvrActor(data, actor.ID)
						}
						break sceneLoop
					}
				}
			}

		}
	}
	tlog.Info("Reverse actor match from XBVR to Stashdb completed")
}

// links an aka group Actor in xbvr to stashdb, based on any links to stashdb by actors in the group
// it then adds links for other actors in the group that don't have links
func LinkOnXbvrAkaGroups() {
	log.Infof("LinkActors based on XBR Aka Groups")
	db, _ := models.GetDB()
	defer db.Close()

	// Link Aka group actors
	var unlinkedAkaActors []models.Actor
	db.Where("name like 'aka:%' and IFNULL(image_url, '') = ''").Find(&unlinkedAkaActors)
	for _, akaActor := range unlinkedAkaActors {
		var akaGroup models.Aka
		db.Preload("Akas").
			Where("aka_actor_id = ?", akaActor.ID).
			First(&akaGroup)

		for _, actor := range akaGroup.Akas {
			var extref models.ExternalReference
			db.
				Table("external_reference_links").
				Joins("JOIN external_references on external_references.id = external_reference_links.external_reference_id").
				Preload("XbvrLinks").
				Where("internal_db_id = ? and external_reference_links.external_source='stashdb performer'", actor.ID).
				Select("external_references.*").
				First(&extref)
			if extref.ID != 0 {
				var data models.StashPerformer
				json.Unmarshal([]byte(extref.ExternalData), &data)
				xbrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: akaActor.ID, InternalNameId: akaActor.Name,
					ExternalReferenceID: extref.ID, ExternalSource: extref.ExternalSource, ExternalId: extref.ExternalId, MatchType: 60}
				extref.XbvrLinks = append(extref.XbvrLinks, xbrLink)
				extref.Save()
				UpdateXbvrActor(data, akaActor.ID)
				break
			}
		}
	}

	// Link unlinked actors in aka group
	var akaGroup []models.Aka
	db.Preload("Akas").
		Joins("JOIN external_reference_links on external_reference_links.internal_db_id = akas.aka_actor_id and external_reference_links.external_source='stashdb performer'").
		Find(&akaGroup)
	for _, akaActor := range akaGroup {
		var akaActorRef models.ExternalReference
		db.Table("external_reference_links").
			Preload("XbvrLinks").
			Joins("JOIN external_references on external_references.id = external_reference_links.external_reference_id").
			Where("internal_db_id = ? and external_reference_links.external_source='stashdb performer'", akaActor.AkaActorId).
			Select("external_references.*").
			First(&akaActorRef)
		var akaActorStashPerformer models.StashPerformer
		json.Unmarshal([]byte(akaActorRef.ExternalData), &akaActorStashPerformer)

		for _, actor := range akaActor.Akas {
			var extref models.ExternalReference
			db.Table("external_reference_links").
				Joins("JOIN external_references on external_references.id = external_reference_links.external_reference_id").
				Where("internal_db_id = ? and external_reference_links.external_source='stashdb performer'", actor.ID).
				Select("external_references.*").
				First(&extref)
			if extref.ID == 0 {
				xbrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name,
					ExternalReferenceID: akaActorRef.ID, ExternalSource: akaActorRef.ExternalSource, ExternalId: akaActorRef.ExternalId, MatchType: 70}
				akaActorRef.XbvrLinks = append(akaActorRef.XbvrLinks, xbrLink)
				akaActorRef.Save()
				UpdateXbvrActor(akaActorStashPerformer, actor.ID)
			}

		}
	}
}
