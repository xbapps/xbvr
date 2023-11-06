package scrape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
)

type Site struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Regex       string `json:"regex"`
	ValidTypes  string `json:"valid_types"`
}

type QueryScenesResponse struct {
	Count  int                 `json:"count"`
	Scenes []models.StashScene `json:"scenes"`
}

type QueryScenesData struct {
	QueryScenes QueryScenesResponse `json:"queryScenes"`
}

type QueryScenesResult struct {
	Data QueryScenesData `json:"data"`
}
type FindStudioResponse struct {
	Studio models.StashStudio `json:"studio"`
}

type FindStudioData struct {
	Studio models.StashStudio `json:"findStudio"`
}
type FindStudioResult struct {
	Data FindStudioData `json:"data"`
}

type FindPerformerResult struct {
	Data FindPerformerData `json:"data"`
}
type FindPerformerData struct {
	Performer models.StashPerformer `json:"findPerformer"`
}

type QueryPerformerResult struct {
	Data QueryPerformerResultTypeData `json:"data"`
}
type QueryPerformerResultTypeData struct {
	QueryPerformers QueryPerformerResultType `json:"queryPerformers"`
}
type QueryPerformerResultType struct {
	Count      int                     `json:"count"`
	Performers []models.StashPerformer `json:"performers"`
}
type Image struct {
	ID     string `json:"id"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

var Config models.ActorScraperConfig

func StashDb() {
	if config.Config.Advanced.StashApiKey == "" {
		return
	}
	tlog := log.WithField("task", "scrape")
	scraperID := "stashdb"
	siteID := "stashdb"
	logScrapeStart(scraperID, siteID)

	var sites []models.Site
	db, _ := models.GetDB()
	defer db.Close()

	Config = models.BuildActorScraperRules()
	db.Where(&models.Site{IsEnabled: true}).Order("id").Find(&sites)

	for _, site := range sites {
		tlog.Infof("Scraping stash studio %s", site.Name)
		sitename := site.Name
		if i := strings.Index(sitename, " ("); i != -1 {
			sitename = sitename[:i]
		}
		studio := findStudio(sitename, "name")

		// check for a config entry if site not found
		if studio.Data.Studio.ID == "" && Config.StashSceneMatching[site.ID].StashId != "" {
			studio = findStudio(Config.StashSceneMatching[site.ID].StashId, "id")
		}

		if studio.Data.Studio.ID != "" {
			siteConfig := Config.StashSceneMatching[site.ID]
			var ext models.ExternalReference
			ext.FindExternalId("stashdb studio", studio.Data.Studio.ID)
			if ext.ID == 0 || studio.Data.Studio.Updated.UTC().Sub(ext.ExternalDate.UTC()).Seconds() > 1 {
				jsonData, _ := json.MarshalIndent(studio.Data.Studio, "", "  ")
				ext := models.ExternalReference{ExternalSource: "stashdb studio", ExternalURL: "https://stashdb.org/studios/" + studio.Data.Studio.ID,
					ExternalId: studio.Data.Studio.ID, ExternalDate: studio.Data.Studio.Updated, ExternalData: string(jsonData),
					XbvrLinks: []models.ExternalReferenceLink{{InternalTable: "sites", InternalNameId: site.ID, ExternalSource: "stashdb studio", ExternalId: studio.Data.Studio.ID}}}
				ext.AddUpdateWithId()
			}
			processStudioPerformers(studio.Data.Studio.ID)
			scenes := getScenes(studio.Data.Studio.ID, siteConfig.ParentId, siteConfig.TagIdFilter)
			saveScenesToExternalReferences(scenes, studio.Data.Studio.ID)
		} else {
			log.Infof("No Stash Studio matching %v", site.Name)
		}
		tlog.Info("Scrape of Stashdb completed")
	}
}

func findStudio(studio string, field string) FindStudioResult {
	fieldType := "String"
	if field == "id" {
		fieldType = "ID"
	}

	query := `
	query  findStudio($` + field + `: ` + fieldType + `) {
		findStudio(` + field + `: $` + field + `) {
			id
			name
		  	parent {
				name
				id
			}
			updated
		}
	}
	`

	// Define the variables needed for your query as a Go map
	variables := `{"` + field + `": "` + studio + `"}`

	resp := callStashDb(query, variables)
	var data FindStudioResult
	json.Unmarshal(resp, &data)
	return data
}

func processStudioPerformers(studioId string) {
	page := 1
	performerList := getPerformersPage(studioId, page)
	for len(performerList.Data.QueryPerformers.Performers) < performerList.Data.QueryPerformers.Count {
		page += 1
		nextList := getPerformersPage(studioId, page)
		if len(nextList.Data.QueryPerformers.Performers) == 0 {
			log.Info("error")
			page = page - 1
		}
		performerList.Data.QueryPerformers.Performers = append(performerList.Data.QueryPerformers.Performers, nextList.Data.QueryPerformers.Performers...)
	}

	for _, performer := range performerList.Data.QueryPerformers.Performers {
		updatePerformer(performer)
	}
}
func getPerformersPage(studioId string, page int) QueryPerformerResult {
	query := `
		query  queryPerformers($input: PerformerQueryInput!) {
			queryPerformers(input: $input) {
				count
					performers {
						id
						updated
						gender
						name
					}		  
				}
			}`
	// Define the variables needed for your query as a Go map
	variables := `
		{"input":{
			"studio_id": "` + studioId + `",
			"page": ` + strconv.Itoa(page) + `,
			"per_page": 100
			}
		}
		`

	resp := callStashDb(query, variables)
	var data QueryPerformerResult
	json.Unmarshal(resp, &data)
	return data

}
func getScenes(studioId string, parentId string, tagId string) QueryScenesResult {
	// find the most recent scene from the database
	db, _ := models.GetDB()
	defer db.Close()
	var lastUpdate models.ExternalReference
	db.Where("external_source = ? and external_data like ?", "stashdb scene", "%"+studioId+"%").Order("external_date DESC").First(&lastUpdate)
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
	sceneList = getScenePage(variables)
	nextList = sceneList
	for len(nextList.Data.QueryScenes.Scenes) > 0 &&
		len(sceneList.Data.QueryScenes.Scenes) < sceneList.Data.QueryScenes.Count && // {
		lastUpdate.ExternalDate.Before(sceneList.Data.QueryScenes.Scenes[len(sceneList.Data.QueryScenes.Scenes)-1].Studio.Updated) {
		page += 1
		if parentId != "" {
			variables = getParentSceneQueryVariable(parentId, tagId, page, count)
		} else {
			variables = getStudioSceneQueryVariable(studioId, page, count)
		}
		nextList = getScenePage(variables)
		sceneList.Data.QueryScenes.Scenes = append(sceneList.Data.QueryScenes.Scenes, nextList.Data.QueryScenes.Scenes...)
	}
	return sceneList
}

// Builds a query variable to get scenes from the Studio
func getStudioSceneQueryVariable(studioId string, page int, count int) string {
	return `
	{"input":{
				"studios": {
					"modifier": "EQUALS",
					"value": "` + studioId + `"
				},
				"page": ` + strconv.Itoa(page) + `,
				"per_page": ` + strconv.Itoa(count) + `,
				"sort": "UPDATED_AT"
			}
		}`

}

// Builds a query variable to get scenes from the Parent Studio
// Uses the tagId to filter just scenes tag as Virtual Reality
func getParentSceneQueryVariable(parentId string, tagId string, page int, count int) string {
	return `
	{"input":{
		"tags": {					
			"value": "` + tagId + `",
			"modifier": "INCLUDES"
		},
		"parentStudio": "` + parentId + `",				 
		
		"page": ` + strconv.Itoa(page) + `,
		"per_page": ` + strconv.Itoa(count) + `,
		"sort": "UPDATED_AT"
		}
	}
`
}

// calls graphql scene query and return a list of scenes
func getScenePage(variables string) QueryScenesResult {
	query := `
	query  queryScenes($input: SceneQueryInput!) {
		queryScenes(input: $input) {
		  count
		 scenes{
		  id
		  title
		  details
		  release_date
		  date
		  updated
		  urls{
			url
			type
			site {
                id
                name
                description
                url
                regex
                valid_types
            }
		  }
		  studio{
			  id
			  name
			  updated
		  }
		  images{
			url
			width
			height
		  }
		  performers{
			performer{
			  id
			  updated
			  gender
			  name
			  aliases
			}
			as
		  }
		  fingerprints{
			hash
			duration
			submissions
		  }
		  duration
		  code  
		  deleted
		}
	  }
	  }
	  `

	// Define the variables needed for your query as a Go map
	resp := callStashDb(query, variables)
	var data QueryScenesResult
	json.Unmarshal(resp, &data)
	return data
}

func saveScenesToExternalReferences(scenes QueryScenesResult, studioId string) {
	tlog := log.WithField("task", "scrape")
	startTime := time.Now()
	nextProgressTime := startTime.Add(1 * time.Minute)

	db, _ := models.GetDB()
	defer db.Close()

	// loop in reverse, we only get scenes since the last update, so we must process from the oldest to the newest
	// in case the user shuts down while processing
	for idx := len(scenes.Data.QueryScenes.Scenes) - 1; idx >= 0; idx-- {
		scene := scenes.Data.QueryScenes.Scenes[idx]
		var xbvrId uint
		var xbvrSceneId string

		// have to set the Studio in case the scene is from a Parent Studio with a different Id
		scene.Studio.ID = studioId
		// check if it's time to print a progress message
		if time.Now().After(nextProgressTime) {
			tlog.Infof("Processing scene %v or %v for StashDb %s", len(scenes.Data.QueryScenes.Scenes)-idx, len(scenes.Data.QueryScenes.Scenes), scene.Studio.Name)
			nextProgressTime = nextProgressTime.Add(1 * time.Minute)
		}
		var existingRef models.ExternalReference
		existingRef.FindExternalId("stashdb scene", scene.ID)
		if existingRef.ID != 0 && scene.Updated.UTC().Sub(existingRef.ExternalDate.UTC()).Seconds() < 1 {
			continue
		}

		jsonData, _ := json.MarshalIndent(scene, "", "  ")

		// chek if we have the performers, may not in the case of loading scenes from the parent studio
		for _, performer := range scene.Performers {
			updatePerformer(performer.Performer)
		}

		// see if we can link to an xbvr scene based on the urls
		for _, url := range scene.URLs {
			if url.Type == "STUDIO" {
				var xbvrScene models.Scene
				url_no_slash := strings.TrimRight(url.URL, "/")
				db.Where("scene_url like ? or scene_url like ?", url_no_slash, url_no_slash+"/").Preload("Cast").Find(&xbvrScene)
				if xbvrScene.ID != 0 {
					xbvrId = xbvrScene.ID
					xbvrSceneId = xbvrScene.SceneID
				}
			}
		}

		var xbrLink []models.ExternalReferenceLink
		if xbvrId != 0 {
			xbrLink = append(xbrLink, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: xbvrId, InternalNameId: xbvrSceneId, ExternalSource: "stashdb scene", ExternalId: scene.ID, MatchType: 10})
		}
		ext := models.ExternalReference{ExternalSource: "stashdb scene", ExternalURL: "https://stashdb.org/scenes/" + scene.ID, ExternalId: scene.ID, ExternalDate: scene.Updated, ExternalData: string(jsonData),
			XbvrLinks: xbrLink}
		ext.AddUpdateWithId()
	}
}

func updatePerformer(newPerformer models.StashPerformer) {
	var ext models.ExternalReference
	ext.FindExternalId("stashdb performer", newPerformer.ID)
	var oldPerformer models.StashPerformer
	json.Unmarshal([]byte(ext.ExternalData), &oldPerformer)
	if ext.ID == 0 || newPerformer.Updated.UTC().Sub(oldPerformer.Updated.UTC()).Seconds() > 1 {
		fullDetails := getStashPerformer(newPerformer.ID).Data.Performer
		jsonData, _ := json.MarshalIndent(fullDetails, "", "  ")
		newext := models.ExternalReference{ExternalSource: "stashdb performer", ExternalURL: "https://stashdb.org/performers/" + fullDetails.ID, ExternalId: fullDetails.ID, ExternalDate: fullDetails.Updated, ExternalData: string(jsonData)}
		if ext.ID != 0 {
			newext.XbvrLinks = ext.XbvrLinks
		}
		newext.AddUpdateWithId()
		for _, link := range newext.XbvrLinks {
			externalreference.UpdateXbvrActor(fullDetails, link.InternalDbId)
		}

	}
}

func RefreshPerformer(performerId string) {
	if config.Config.Advanced.StashApiKey == "" {
		return
	}
	var ext models.ExternalReference
	ext.FindExternalId("stashdb performer", performerId)
	fullDetails := getStashPerformer(performerId).Data.Performer
	if fullDetails.ID == "" {
		return
	}
	jsonData, _ := json.MarshalIndent(fullDetails, "", "  ")
	newext := models.ExternalReference{ExternalSource: "stashdb performer", ExternalURL: "https://stashdb.org/performers/" + fullDetails.ID, ExternalId: fullDetails.ID, ExternalDate: fullDetails.Updated, ExternalData: string(jsonData)}
	if ext.ID != 0 {
		newext.XbvrLinks = ext.XbvrLinks
	}
	newext.AddUpdateWithId()
	for _, link := range newext.XbvrLinks {
		externalreference.UpdateXbvrActor(fullDetails, link.InternalDbId)
	}
}

func getStashPerformer(performer string) FindPerformerResult {

	query := `
	query  findPerformer($id: ID!) {
		findPerformer(id: $id) {
  id
  name
  disambiguation
  aliases
  gender
  urls{
      url
      type
      site {
          id
          name
          description
          regex
          valid_types
      }
  }
  birth_date
  age
  ethnicity
  country
  eye_color
  hair_color
  height
  cup_size
  band_size
  waist_size
  hip_size
  breast_type
  career_start_year
  career_end_year
  tattoos { 
      description
      location
       }
  piercings { 
      description
      location
       }
  images{
      id
      url
      width
      height
      }
  
  deleted
  merged_ids
  created
  updated

	  }
	  }
	`

	// Define the variables needed for your query as a Go map
	var data FindPerformerResult
	variables := `{"id": "` + performer + `"}`
	resp := callStashDb(query, variables)
	err := json.Unmarshal(resp, &data)
	if err != nil {
		log.Errorf("Eror extracting actor json")
	}
	return data
}
func callStashDb(query string, rawVariables string) []byte {
	var variables map[string]interface{}
	json.Unmarshal([]byte(rawVariables), &variables)

	// Convert the variables map to JSON
	jsonVariables, _ := json.Marshal(variables)

	// Create an HTTP POST request to send the GraphQL query to the endpoint
	req, err := http.NewRequest("POST", "http://stashdb.org/graphql", bytes.NewBuffer([]byte(fmt.Sprintf(`{"query":%q,"variables":%s}`, query, jsonVariables))))
	if err != nil {
		log.Infof("error geting new request in callStashDb %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("ApiKey", config.Config.Advanced.StashApiKey)

	callClient := func() []byte {
		var bodyBytes []byte
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Infof("error client.do  in callStashDb %s", err)
		}

		defer resp.Body.Close()

		bodyBytes, _ = io.ReadAll(resp.Body)
		return bodyBytes
	}
	return callClient()
}
