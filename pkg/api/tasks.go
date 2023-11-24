package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type RequestScrapeJAVR struct {
	Scraper string `json:"s"`
	Query   string `json:"q"`
}

type RequestScrapeTPDB struct {
	ApiToken string `json:"apiToken"`
	SceneUrl string `json:"sceneUrl"`
}

type RequestSingleScrape struct {
	Site           string                            `json:"site"`
	SceneUrl       string                            `json:"sceneurl"`
	AdditionalInfo []RequestSingleScrapeAdditionInfo `json:"additionalinfo"`
}

type RequestSingleScrapeAdditionInfo struct {
	FieldName   string `json:"fieldName"`
	FieldPrompt string `json:"fieldPrompt"`
	Placeholder string `json:"placeholder"`
	FieldValue  string `json:"fieldValue"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}
type ResponseBackupBundle struct {
	Response string `json:"status"`
}

type ResponseSceneScrape struct {
	Response string       `json:"status"`
	Scene    models.Scene `json:"scene"`
}

type TaskResource struct{}

func (i TaskResource) WebService() *restful.WebService {
	tags := []string{"Task"}

	ws := new(restful.WebService)

	ws.Path("/api/task").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/rescan").To(i.rescan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/rescan/{storage-id}").To(i.rescan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scene-refresh").To(i.sceneRrefresh).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/clean-tags").To(i.cleanTags).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scrape").To(i.scrape).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/singlescrape").To(i.singleScrape).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseSceneScrape{}))

	ws.Route(ws.GET("/index").To(i.index).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/preview/generate").To(i.previewGenerate).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/funscript/export-all").To(i.exportAllFunscripts).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/funscript/export-new").To(i.exportNewFunscripts).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/bundle/backup").To(i.backupBundle).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseBackupBundle{}))

	ws.Route(ws.POST("/bundle/restore").To(i.restoreBundle).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scrape-javr").To(i.scrapeJAVR).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scrape-tpdb").To(i.scrapeTPDB).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i TaskResource) rescan(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("storage-id"))
	if err != nil {
		// no storage-id, refresh all
		go tasks.RescanVolumes(-1)
		return
	} else {
		// just refresh the specified path
		go tasks.RescanVolumes(id)
	}
}

func (i TaskResource) sceneRrefresh(req *restful.Request, resp *restful.Response) {
	go tasks.RefreshSceneStatuses()
}

func (i TaskResource) cleanTags(req *restful.Request, resp *restful.Response) {
	go tasks.CleanTags()
}

func (i TaskResource) index(req *restful.Request, resp *restful.Response) {
	go tasks.SearchIndex()
}

func (i TaskResource) scrape(req *restful.Request, resp *restful.Response) {
	qSiteID := req.QueryParameter("site")
	if qSiteID == "" {
		qSiteID = "_enabled"
	}
	go tasks.Scrape(qSiteID, "", "")
}
func (i TaskResource) singleScrape(req *restful.Request, resp *restful.Response) {
	var scrapeParams RequestSingleScrape
	req.ReadEntity(&scrapeParams)
	additionalInfo, _ := json.Marshal(scrapeParams.AdditionalInfo)

	newScene := tasks.ScrapeSingleScene(scrapeParams.Site, scrapeParams.SceneUrl, string(additionalInfo))

	createResp := &ResponseSceneScrape{
		Response: "OK",
		Scene:    newScene,
	}
	resp.WriteHeaderAndEntity(http.StatusOK, createResp)
}

func (i TaskResource) exportAllFunscripts(req *restful.Request, resp *restful.Response) {
	tasks.ExportFunscripts(resp.ResponseWriter, false)
}

func (i TaskResource) exportNewFunscripts(req *restful.Request, resp *restful.Response) {
	tasks.ExportFunscripts(resp.ResponseWriter, true)
}

func (i TaskResource) backupBundle(req *restful.Request, resp *restful.Response) {
	inclAllSites, _ := strconv.ParseBool(req.QueryParameter("allSites"))
	onlyIncludeOfficalSites, _ := strconv.ParseBool(req.QueryParameter("onlyIncludeOfficalSites"))
	inclScenes, _ := strconv.ParseBool(req.QueryParameter("inclScenes"))
	inclFileLinks, _ := strconv.ParseBool(req.QueryParameter("inclLinks"))
	inclCuepoints, _ := strconv.ParseBool(req.QueryParameter("inclCuepoints"))
	inclHistory, _ := strconv.ParseBool(req.QueryParameter("inclHistory"))
	inclPlaylists, _ := strconv.ParseBool(req.QueryParameter("inclPlaylists"))
	inclActorAkas, _ := strconv.ParseBool(req.QueryParameter("inclActorAkas"))
	inclTagGroups, _ := strconv.ParseBool(req.QueryParameter("inclTagGroups"))
	inclVolumes, _ := strconv.ParseBool(req.QueryParameter("inclVolumes"))
	inclSites, _ := strconv.ParseBool(req.QueryParameter("inclSites"))
	inclActions, _ := strconv.ParseBool(req.QueryParameter("inclActions"))
	inclExtRefs, _ := strconv.ParseBool(req.QueryParameter("inclExtRefs"))
	inclActors, _ := strconv.ParseBool(req.QueryParameter("inclActors"))
	inclActorActions, _ := strconv.ParseBool(req.QueryParameter("inclActorActions"))
	inclConfig, _ := strconv.ParseBool(req.QueryParameter("inclConfig"))
	playlistId := req.QueryParameter("playlistId")
	download := req.QueryParameter("download")

	bundle := tasks.BackupBundle(inclAllSites, onlyIncludeOfficalSites, inclScenes, inclFileLinks, inclCuepoints, inclHistory, inclPlaylists, inclActorAkas, inclTagGroups, inclVolumes, inclSites, inclActions, inclExtRefs, inclActors, inclActorActions, inclConfig, playlistId, "", "")
	if download == "true" {
		resp.WriteHeaderAndEntity(http.StatusOK, ResponseBackupBundle{Response: "Ready to Download from http://xxx.xxx.xxx.xxx:9999/download/xbvr-content-bundle.json"})
	} else {
		// not downloading, display the bundle data
		resp.WriteHeaderAndEntity(http.StatusOK, (bundle))
	}

}

func (i TaskResource) restoreBundle(req *restful.Request, resp *restful.Response) {
	var r tasks.RequestRestore

	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	go tasks.RestoreBundle(r)
}

func (i TaskResource) previewGenerate(req *restful.Request, resp *restful.Response) {
	go tasks.GeneratePreviews(nil)
}

func (i TaskResource) scrapeJAVR(req *restful.Request, resp *restful.Response) {
	var r RequestScrapeJAVR
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.Query != "" {
		go tasks.ScrapeJAVR(r.Query, r.Scraper)
	}
}

func (i TaskResource) scrapeTPDB(req *restful.Request, resp *restful.Response) {
	var r RequestScrapeTPDB
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.ApiToken != "" && r.SceneUrl != "" {
		go tasks.ScrapeTPDB(strings.TrimSpace(r.ApiToken), strings.TrimSpace(r.SceneUrl))
	}
}
