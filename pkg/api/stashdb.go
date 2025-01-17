package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

func (i ExternalReference) refreshStashPerformer(req *restful.Request, resp *restful.Response) {
	performerId := req.PathParameter("performerid")
	scrape.RefreshPerformer(performerId)
	resp.WriteHeader(http.StatusOK)
}

func (i ExternalReference) stashSceneApplyRules(req *restful.Request, resp *restful.Response) {
	go externalreference.ApplySceneRules()
}

func (i ExternalReference) matchAkaPerformers(req *restful.Request, resp *restful.Response) {
	go externalreference.MatchAkaPerformers()

}
func (i ExternalReference) stashDbUpdateData(req *restful.Request, resp *restful.Response) {
	go externalreference.UpdateAllPerformerData()

}
func (i ExternalReference) stashRunAll(req *restful.Request, resp *restful.Response) {
	StashdbRunAll()
}
func (i ExternalReference) linkScene2Stashdb(req *restful.Request, resp *restful.Response) {
	sceneId := req.PathParameter("scene-id")
	stashdbId := req.PathParameter("stashdb-id")
	stashdbId = strings.TrimPrefix(stashdbId, "https://stashdb.org/scenes/")
	var scene models.Scene

	db, _ := models.GetDB()
	defer db.Close()

	if strings.Contains(sceneId, "-") {
		scene.GetIfExist(sceneId)
	} else {
		id, _ := strconv.Atoi(req.PathParameter("scene-id"))
		scene.GetIfExistByPK(uint(id))
	}
	if scene.ID == 0 {
		return
	}
	stashScene := scrape.GetStashDbScene(stashdbId)

	var existingRef models.ExternalReference
	existingRef.FindExternalId("stashdb scene", stashdbId)

	jsonData, _ := json.MarshalIndent(stashScene.Data.Scene, "", "  ")

	// chek if we have the performers, may not in the case of loading scenes from the parent studio
	for _, performer := range stashScene.Data.Scene.Performers {
		scrape.UpdatePerformer(performer.Performer)
	}

	var xbrLink []models.ExternalReferenceLink
	xbrLink = append(xbrLink, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: scene.ID, InternalNameId: scene.SceneID, ExternalSource: "stashdb scene", ExternalId: stashdbId, MatchType: 5})
	ext := models.ExternalReference{ExternalSource: "stashdb scene", ExternalURL: "https://stashdb.org/scenes/" + stashdbId, ExternalId: stashdbId, ExternalDate: time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC), ExternalData: string(jsonData),
		XbvrLinks: xbrLink}
	ext.AddUpdateWithId()

	// check for actor not yet linked
	for _, actor := range scene.Cast {
		var extreflinks []models.ExternalReferenceLink
		db.Preload("ExternalReference").Where(&models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, ExternalSource: "stashdb performer"}).Find(&extreflinks)
		if len(extreflinks) == 0 {
			stashPerformerId := ""
			for _, stashPerf := range stashScene.Data.Scene.Performers {
				if strings.EqualFold(stashPerf.Performer.Name, actor.Name) || strings.EqualFold(stashPerf.As, actor.Name) {
					stashPerformerId = stashPerf.Performer.ID
					continue
				}
				for _, alias := range stashPerf.Performer.Aliases {
					if strings.EqualFold(alias, actor.Name) {
						stashPerformerId = stashPerf.Performer.ID
					}
				}
			}
			if stashPerformerId != "" {
				scrape.RefreshPerformer(stashPerformerId)
				var actorRef models.ExternalReference
				actorRef.FindExternalId("stashdb performer", stashPerformerId)
				var performer models.StashPerformer
				json.Unmarshal([]byte(actorRef.ExternalData), &performer)

				xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 90,
					ExternalReferenceID: actorRef.ID, ExternalSource: actorRef.ExternalSource, ExternalId: actorRef.ExternalId}
				actorRef.XbvrLinks = append(actorRef.XbvrLinks, xbvrLink)
				actorRef.AddUpdateWithId()

				externalreference.UpdateXbvrActor(performer, actor.ID)
			}
		}
	}

	// reread the scene to return updated data
	scene.GetIfExistByPK(scene.ID)
	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i ExternalReference) searchForStashdbScene(req *restful.Request, resp *restful.Response) {
	query := req.QueryParameter("q")

	var warnings []string
	type StashSearchScenePerformerResult struct {
		Name string
		Url  string
	}
	type StashSearchSceneResult struct {
		Url         string
		ImageUrl    string
		Performers  []StashSearchScenePerformerResult
		Title       string
		Studio      string
		Duration    string
		Description string
		Weight      int
		Date        string
		Id          string
	}
	type StashSearchSceneResponse struct {
		Status  string
		Results []StashSearchSceneResult
	}
	results := make(map[string]StashSearchSceneResult)

	sceneId := req.PathParameter("scene-id")
	var scene models.Scene

	db, _ := models.GetDB()
	defer db.Close()

	if strings.Contains(sceneId, "-") {
		scene.GetIfExist(sceneId)
	} else {
		id, _ := strconv.Atoi(req.PathParameter("scene-id"))
		scene.GetIfExistByPK(uint(id))
	}
	if scene.ID == 0 {
		var response StashSearchSceneResponse
		response.Results = []StashSearchSceneResult{}
		response.Status = "XBVR Scene not found"
		resp.WriteHeaderAndEntity(http.StatusOK, response)
		return
	}

	setupStashSearchResult := func(stashScene models.StashScene, weight int) StashSearchSceneResult {
		//common function to call to setup stash response details
		result := StashSearchSceneResult{Id: stashScene.ID, Url: "https://stashdb.org/scenes/" + stashScene.ID, Weight: weight, Title: stashScene.Title, Description: stashScene.Details, Date: stashScene.Date, Studio: stashScene.Studio.Name}
		if len(stashScene.Images) > 0 {
			result.ImageUrl = stashScene.Images[0].URL
		}
		for _, perf := range stashScene.Performers {
			result.Performers = append(result.Performers, StashSearchScenePerformerResult{Name: perf.Performer.Name, Url: `https://stashdb.org/performers/` + perf.Performer.ID})
		}
		if stashScene.Duration > 0 {
			hours := stashScene.Duration / 3600 // calculate hours
			stashScene.Duration %= 3600         // remaining seconds after hours
			minutes := stashScene.Duration / 60 // calculate minutes
			stashScene.Duration %= 60           // remaining seconds after minutes

			// Format the time string
			result.Duration = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, stashScene.Duration)
		}
		return result
	}

	var guidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	idTest := strings.TrimPrefix(strings.TrimSpace(query), "https://stashdb.org/scenes/")

	if guidRegex.MatchString(idTest) {
		stashScene := scrape.GetStashDbScene(idTest)
		if stashScene.Data.Scene.ID != "" {
			results[stashScene.Data.Scene.ID] = setupStashSearchResult(stashScene.Data.Scene, 10000)
			var response StashSearchSceneResponse
			response.Results = []StashSearchSceneResult{results[stashScene.Data.Scene.ID]}
			response.Status = ""
			resp.WriteHeaderAndEntity(http.StatusOK, response)
			return
		}
	}

	stashStudioIds := findStashStudioIds(scene.ScraperId)
	if len(stashStudioIds) == 0 {
		var response StashSearchSceneResponse
		response.Results = []StashSearchSceneResult{}
		response.Status = "Cannot find Stashdb Studio"
		resp.WriteHeaderAndEntity(http.StatusOK, response)
		return
	}

	var xbvrperformers []string
	for _, actor := range scene.Cast {
		var stashlinks []models.ExternalReferenceLink
		db.Preload("ExternalReference").Where(&models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, ExternalSource: "stashdb performer"}).Find(&stashlinks)
		if len(stashlinks) == 0 {
			warnings = append(warnings, actor.Name+" is not linked to Stashdb")
		} else {
			for _, stashPerformer := range stashlinks {
				xbvrperformers = append(xbvrperformers, `"`+stashPerformer.ExternalId+`"`)
			}
		}
	}

	// define a function to update the results found
	scoreResults := func(stashScenes scrape.QueryScenesResult, weightIncrement int, performers []string, studios []string) {
		for _, stashscene := range stashScenes.Data.QueryScenes.Scenes {
			// consider adding weight bump for duration and date
			scoreBump := 0
			if stashscene.Date == scene.ReleaseDateText {
				scoreBump += 15
			}
			if stashscene.Title == scene.Title {
				scoreBump += 25
			}
			if stashscene.Duration > 0 && scene.Duration > 0 {
				stashDur := float64((stashscene.Duration / 60) - scene.Duration)
				if math.Abs(stashDur) <= 2 {
					scoreBump += 5 * int(3-math.Abs(stashDur))
				}
			}
			// check duration from video files
			for _, file := range scene.Files {
				if file.Type == "video" {
					diff := file.VideoDuration - float64(stashscene.Duration)
					if math.Abs(diff) <= 2 {
						scoreBump += 5 * int(3-math.Abs(diff))
					}
				}
			}

			// check it is from a studio we expect
			for _, studio := range studios {
				if strings.ReplaceAll(studio, `"`, ``) == stashscene.Studio.ID {
					scoreBump += 20
				}
			}

			foundActorBump := -5
			for _, sp := range stashscene.Performers {
				for _, xp := range performers {
					if strings.Contains(xp, sp.Performer.ID) {
						if sp.Performer.Gender == "FEMALE" {
							foundActorBump += 10
						} else {
							foundActorBump += 5
						}
					}
				}
			}
			// we have checked if stash performers match xbvr, now check for xbvr performers not matched in stash
			for _, xp := range xbvrperformers {
				for _, sp := range stashscene.Performers {
					if strings.Contains(xp, sp.Performer.ID) {
						if sp.Performer.Gender == "FEMALE" {
							foundActorBump += 15
						} else {
							foundActorBump += 5
						}
					}
				}
			}
			// check actor matches using names and aliases
			for _, actor := range scene.Cast {
				for _, sp := range stashscene.Performers {
					if strings.EqualFold(actor.Name, sp.Performer.Name) || strings.EqualFold(actor.Name, sp.As) {
						if sp.Performer.Gender == "FEMALE" {
							foundActorBump += 15
						} else {
							foundActorBump += 5
						}
						continue
					}
					// try aliases
					for _, alias := range sp.Performer.Aliases {
						if strings.EqualFold(alias, actor.Name) {
							if sp.Performer.Gender == "FEMALE" {
								foundActorBump += 15
							} else {
								foundActorBump += 5
							}
							continue
						}
					}
				}
			}
			if mapEntry, exists := results[stashscene.ID]; exists {
				mapEntry.Weight += weightIncrement + scoreBump + foundActorBump
				results[stashscene.ID] = mapEntry
			} else {
				results[stashscene.ID] = setupStashSearchResult(stashscene, weightIncrement+scoreBump+foundActorBump)
			}
		}
	}

	var fingerprints []string
	for _, file := range scene.Files {
		if file.Type == "video" {
			file.OsHash = "00000000000000000" + file.OsHash
			fingerprints = append(fingerprints, `"`+file.OsHash[len(file.OsHash)-16:]+`"`)
		}
	}
	if len(fingerprints) > 0 {
		fingerprintList := strings.Join(fingerprints, ",")
		fingerprintQuery := `
		{"input":{
					"page": 1,
					"per_page": 150,
					"sort": "UPDATED_AT",
					"fingerprints": {"value": [` +
			fingerprintList +
			`], "modifier":"EQUALS"}
				}
			}`
		stashScenes := scrape.GetScenePage(fingerprintQuery)
		scoreResults(stashScenes, 400, xbvrperformers, stashStudioIds)
	}

	stashScenes := scrape.QueryScenesResult{}
	for _, studio := range stashStudioIds {
		// Exact Title submatch
		titleQuery := `
		{"input":{
					"parentStudio": ` + studio + `,
					"page": 1,
					"per_page": 150,
					"sort": "UPDATED_AT",
					"title": "\"` +
			scene.Title + `\""
				}
			}`
		stashScenes = scrape.GetScenePage(titleQuery)
		scoreResults(stashScenes, 150, xbvrperformers, stashStudioIds)
	}

	if len(xbvrperformers) > 0 {
		performerList := strings.Join(xbvrperformers, ",")
		for _, studio := range stashStudioIds {
			performerQuery := `
			{"input":{
						"parentStudio": ` + studio + `,
						"page": 1,
						"per_page": 150,
						"sort": "UPDATED_AT",
						"performers": {"value": [` +
				performerList +
				`], "modifier":"INCLUDES_ALL"}
					}
				}`
			stashScenes = scrape.GetScenePage(performerQuery)
			scoreResults(stashScenes, 200, xbvrperformers, stashStudioIds)
			if len(stashScenes.Data.QueryScenes.Scenes) == 0 {
				performerQuery = strings.ReplaceAll(performerQuery, "INCLUDES_ALL", "INCLUDES")
				stashScenes := scrape.GetScenePage(performerQuery)
				scoreResults(stashScenes, 100, xbvrperformers, stashStudioIds)
			}
		}
	}

	if len(results) == 0 {
		for _, studio := range stashStudioIds {
			// No match yet, try match any words from the title, not likely to find, as this returns too many results
			titleQuery := `
		{"input":{
					"parentStudio": ` + studio + `,
					"page": 1,
					"per_page": 100,
					"sort": "UPDATED_AT",
					"title": "` +
				scene.Title + `"
				}
			}`
			stashScenes = scrape.GetScenePage(titleQuery)
			scoreResults(stashScenes, 150, xbvrperformers, stashStudioIds)
			page := 2
			for i := 101; i < stashScenes.Data.QueryScenes.Count && page <= 5; {
				titleQuery := `
					{"input":{
								"parentStudio": ` + studio + `,
								"page": ` + strconv.Itoa(page) + `,
								"per_page": 100,
								"sort": "UPDATED_AT",
								"title": "` +
					scene.Title + `"
							}
						}`
				stashScenes = scrape.GetScenePage(titleQuery)
				scoreResults(stashScenes, 150, xbvrperformers, stashStudioIds)
				i = i + 100
				page += 1
			}
		}
	}

	if len(results) == 0 {
		warnings = append(warnings, "No Stashdb Scenes Found")
	}
	// sort and limit the number of results
	// Convert map to a slice of key-value pairs
	pairs := make([]StashSearchSceneResult, 0, len(results))
	for _, v := range results {
		pairs = append(pairs, v)
	}
	// Sort the slice by weight in descending order
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Weight > pairs[j].Weight
	})
	// Take the first 100 entries (or less if there are fewer than 100 entries)
	top100 := pairs[:min(len(pairs), 100)]
	var response StashSearchSceneResponse
	response.Results = top100
	response.Status = strings.Join(warnings, ", ")
	resp.WriteHeaderAndEntity(http.StatusOK, response)
}

func (i ExternalReference) linkActor2Stashdb(req *restful.Request, resp *restful.Response) {
	actorId := req.PathParameter("actor-id")
	stashPerformerId := req.PathParameter("stashdb-id")
	stashPerformerId = strings.TrimPrefix(stashPerformerId, "https://stashdb.org/performers/")
	var actor models.Actor

	db, _ := models.GetDB()
	defer db.Close()

	id, _ := strconv.Atoi(actorId)
	if id == 0 {
		actor.GetIfExist(actorId)
	} else {
		actor.GetIfExistByPK(uint(id))
	}
	if actor.ID == 0 {
		return
	}

	scrape.RefreshPerformer(stashPerformerId)
	var actorRef models.ExternalReference
	actorRef.FindExternalId("stashdb performer", stashPerformerId)
	// change the External Date, this is used to find the most recent change and query
	// stash for changes since then. If wew manually load an actor, we may miss other updates
	actorRef.ExternalDate = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)
	actorRef.Save()

	var performer models.StashPerformer
	json.Unmarshal([]byte(actorRef.ExternalData), &performer)

	xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 90,
		ExternalReferenceID: actorRef.ID, ExternalSource: actorRef.ExternalSource, ExternalId: actorRef.ExternalId}
	actorRef.XbvrLinks = append(actorRef.XbvrLinks, xbvrLink)
	actorRef.AddUpdateWithId()

	externalreference.UpdateXbvrActor(performer, actor.ID)

	// reread the actor to return updated data
	actor.GetIfExistByPK(actor.ID)
	resp.WriteHeaderAndEntity(http.StatusOK, actor)
}

func (i ExternalReference) searchForStashdbActor(req *restful.Request, resp *restful.Response) {
	query := req.QueryParameter("q")
	query = strings.TrimSpace(strings.TrimPrefix(query, "aka:"))

	var warnings []string
	type StashSearchPerformerScenesResult struct {
		Title    string
		Id       string
		Url      string
		Duration string
		ImageUrl string
		Studio   string
	}
	type StashSearchPerformerStudioResult struct {
		Name       string
		Id         string
		Url        string
		SceneCount int
		Matched    bool
	}
	type StashSearchPerformerAliasResult struct {
		Alias   string
		Matched bool
	}
	type StashSearchPerformerResult struct {
		Url            string
		Name           string
		Disambiguation string
		Aliases        []StashSearchPerformerAliasResult
		Id             string
		ImageUrl       []string
		DOB            string
		Weight         int
		Studios        []StashSearchPerformerStudioResult
	}
	type StashSearchPerformersResponse struct {
		Status  string
		Results []StashSearchPerformerResult
	}
	results := make(map[string]StashSearchPerformerResult)

	actorId := req.PathParameter("actor-id")
	var actor models.Actor

	db, _ := models.GetDB()
	defer db.Close()

	id, _ := strconv.Atoi(req.PathParameter("actor-id"))
	if id == 0 {
		actor.GetIfExist(actorId)
	} else {
		actor.GetIfExistByPK(uint(id))
	}
	if actor.ID == 0 {
		var response StashSearchPerformersResponse
		response.Results = []StashSearchPerformerResult{}
		response.Status = "XBVR Actor not found"
		resp.WriteHeaderAndEntity(http.StatusOK, response)
		return
	}

	matchedStudios := map[string]struct{}{}
	matchedAlias := map[string]struct{}{}

	setupStashSearchResult := func(stashPerformer models.StashPerformer, weight int) StashSearchPerformerResult {
		//common function to call to setup stash response details
		result := StashSearchPerformerResult{Id: stashPerformer.ID, Url: "https://stashdb.org/performers/" + stashPerformer.ID, Weight: weight, Name: stashPerformer.Name, DOB: stashPerformer.BirthDate, Disambiguation: stashPerformer.Disambiguation}
		for _, image := range stashPerformer.Images {
			result.ImageUrl = append(result.ImageUrl, image.URL)
		}
		for _, studio := range stashPerformer.Studios {
			_, matched := matchedStudios[studio.Studio.ID]
			if matched {
				matched = true
			}
			result.Studios = append(result.Studios, StashSearchPerformerStudioResult{Name: studio.Studio.Name, Id: studio.Studio.ID, Url: `https://stashdb.org/performers/` + stashPerformer.ID + `?studios=` + studio.Studio.ID, SceneCount: studio.SceneCount, Matched: matched})
		}
		for _, alias := range stashPerformer.Aliases {
			_, matched := matchedAlias[strings.ToLower(alias)]
			if matched {
				matched = true
			}
			result.Aliases = append(result.Aliases, StashSearchPerformerAliasResult{Alias: alias, Matched: matched})
		}

		sort.Slice(result.Studios, func(i, j int) bool {
			return result.Studios[i].Name < result.Studios[j].Name
		})
		return result
	}

	var guidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	idTest := strings.TrimPrefix(strings.TrimSpace(query), "https://stashdb.org/performers/")

	if guidRegex.MatchString(idTest) {
		stashPerformer := scrape.GetStashPerformerFull(idTest)
		if stashPerformer.Data.Performer.ID != "" {
			results[stashPerformer.Data.Performer.ID] = setupStashSearchResult(stashPerformer.Data.Performer, 10000)
			//  need to get studios
			var response StashSearchPerformersResponse
			response.Results = []StashSearchPerformerResult{results[stashPerformer.Data.Performer.ID]}
			response.Status = ""
			resp.WriteHeaderAndEntity(http.StatusOK, response)
			return
		}
	}

	if strings.TrimSpace(query) == "" {
		query = actor.Name
	}
	// define a function to update the results found
	scoreResults := func(stashPerformers []models.StashPerformer, weightIncrement int) {
		for _, stashPerformer := range stashPerformers {
			// consider adding weight bump for duration and date
			scoreBump := 0
			lcaseActorName := strings.ToLower(actor.Name)
			if stashPerformer.Name == actor.Name {
				scoreBump += 200
			}
			if strings.Contains(strings.ToLower(stashPerformer.Name), lcaseActorName) {
				scoreBump += 30
			}
			for _, alias := range stashPerformer.Aliases {
				lcaseAlias := strings.ToLower(alias)
				if strings.EqualFold(lcaseAlias, lcaseActorName) || strings.EqualFold(lcaseAlias, strings.ToLower(query)) {
					scoreBump += 50
					matchedAlias[strings.ToLower(alias)] = struct{}{}
				} else {
					if strings.Contains(lcaseAlias, lcaseActorName) || strings.Contains(lcaseActorName, lcaseAlias) || strings.Contains(lcaseAlias, strings.ToLower(query)) {
						scoreBump += 20
						matchedAlias[strings.ToLower(alias)] = struct{}{}
					}
				}
			}
			// dob checks

			if actor.Gender != "" && strings.EqualFold(actor.Gender, stashPerformer.Gender) {
				scoreBump += 10
			}

			var stashExtLink models.ExternalReferenceLink
			commonDb, _ := models.GetCommonDB()
			for _, xbvrScene := range actor.Scenes {
				for _, stashStudio := range stashPerformer.Studios {
					xbvrsite := xbvrScene.Site
					var siteRef models.ExternalReferenceLink
					commonDb.Where(&models.ExternalReferenceLink{InternalTable: "sites", InternalNameId: xbvrScene.ScraperId, ExternalSource: "stashdb studio"}).First(&siteRef)
					if strings.Index(xbvrsite, " (") != -1 {
						xbvrsite = xbvrsite[:strings.Index(xbvrsite, " (")]
					}
					if strings.EqualFold(stashStudio.Studio.Name, xbvrsite) || siteRef.ExternalId == stashStudio.Studio.ID {
						scoreBump += 5
						matchedStudios[stashStudio.Studio.ID] = struct{}{}
					}
				}
				// check if we have a linked scene with this performer
				links := stashExtLink.FindByExternalSource("scenes", xbvrScene.ID, "stashdb scene")
				if len(links) == 0 {
					continue
				}
				for _, link := range links {
					if strings.Contains(link.ExternalReference.ExternalData, stashPerformer.ID) {
						scoreBump += 50
					}
				}
			}

			if mapEntry, exists := results[stashPerformer.ID]; exists {
				mapEntry.Weight += weightIncrement + scoreBump
				results[stashPerformer.ID] = mapEntry
			} else {
				results[stashPerformer.ID] = setupStashSearchResult(stashPerformer, weightIncrement+scoreBump)
			}
		}
	}

	stashPerformers := scrape.SearchPerformerResult{}
	stashPerformers = scrape.SearchStashPerformer(query)
	scoreResults(stashPerformers.Data.Performers, 150)

	if len(results) == 0 {
		warnings = append(warnings, "No Stashdb Performers Found")
	}
	// Sort the results by the weight score and limit to 100
	// Convert map to a slice of key-value pairs
	pairs := make([]StashSearchPerformerResult, 0, len(results))
	for _, v := range results {
		pairs = append(pairs, v)
	}
	// Sort the slice by weight in descending order
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Weight > pairs[j].Weight
	})
	// Take the first 100 entries (or less if there are fewer than 100 entries)
	top100 := pairs[:min(len(pairs), 100)]
	var response StashSearchPerformersResponse
	response.Results = top100
	response.Status = strings.Join(warnings, ", ")
	resp.WriteHeaderAndEntity(http.StatusOK, response)
}

func StashdbRunAll() {
	go func() {
		if !models.CheckLock("scrape") {
			models.CreateLock("scrape")
			defer models.RemoveLock("scrape")

			t0 := time.Now()
			tlog := log.WithField("task", "scrape")
			tlog.Infof("StashDB Refresh started at %s", t0.Format("Mon Jan _2 15:04:05 2006"))
			scrape.StashDb()

			externalreference.ApplySceneRules()
			externalreference.MatchAkaPerformers()
			externalreference.UpdateAllPerformerData()
			tlog = log.WithField("task", "scrape")
			tlog.Infof("Stashdb Refresh Complete in %s",
				time.Since(t0).Round(time.Second))
		}
	}()
}
func findStashStudioIds(scraper string) []string {
	stashIds := map[string]struct{}{}
	var site models.Site
	site.GetIfExist(scraper)

	db, _ := models.GetCommonDB()
	var refs []models.ExternalReferenceLink
	db.Preload("ExternalReference").Where(&models.ExternalReferenceLink{InternalTable: "sites", InternalNameId: scraper, ExternalSource: "stashdb studio"}).Find(&refs)

	for _, site := range refs {
		stashIds[site.ExternalId] = struct{}{}
	}

	config := models.BuildActorScraperRules()
	s := config.StashSceneMatching[scraper]
	for _, value := range s {
		stashIds[value.StashId] = struct{}{}
	}

	if len(stashIds) == 0 {
		// if we don't have any lookup stashdb using the sitename
		sitename := site.Name
		if i := strings.Index(sitename, " ("); i != -1 {
			sitename = sitename[:i]
		}
		studio := scrape.FindStashdbStudio(sitename, "name")
		stashIds[studio.Data.Studio.ID] = struct{}{}
	}
	var results []string
	for key, _ := range stashIds {
		results = append(results, `"`+key+`"`)
	}
	return results
}
