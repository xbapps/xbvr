package api

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-resty/resty/v2"
	"github.com/jinzhu/gorm"
	"github.com/mcuadros/go-version"
	"github.com/pkg/errors"
	"github.com/putdotio/go-putio"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type NewVolumeRequest struct {
	Type  string `json:"type"`
	Path  string `json:"path"`
	Token string `json:"token"`
}

type VersionCheckResponse struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateNotify   bool   `json:"update_notify"`
}

type RequestSaveOptionsWeb struct {
	TagSort           string `json:"tagSort"`
	SceneHidden       bool   `json:"sceneHidden"`
	SceneWatchlist    bool   `json:"sceneWatchlist"`
	SceneFavourite    bool   `json:"sceneFavourite"`
	SceneWishlist     bool   `json:"sceneWishlist"`
	SceneWatched      bool   `json:"sceneWatched"`
	SceneEdit         bool   `json:"sceneEdit"`
	SceneDuration     bool   `json:"sceneDuration"`
	SceneCuepoint     bool   `json:"sceneCuepoint"`
	ShowHspFile       bool   `json:"showHspFile"`
	ShowSubtitlesFile bool   `json:"showSubtitlesFile"`
	SceneTrailerlist  bool   `json:"sceneTrailerlist"`
	ShowScriptHeatmap bool   `json:"showScriptHeatmap"`
	UpdateCheck       bool   `json:"updateCheck"`
}

type RequestSaveOptionsAdvanced struct {
	ShowInternalSceneId   bool   `json:"showInternalSceneId"`
	ShowHSPApiLink        bool   `json:"showHSPApiLink"`
	ShowSceneSearchField  bool   `json:"showSceneSearchField"`
	StashApiKey           string `json:"stashApiKey"`
	ScrapeActorAfterScene bool   `json:"scrapeActorAfterScene"`
	UseImperialEntry      bool   `json:"useImperialEntry"`
}

type RequestSaveOptionsFunscripts struct {
	ScrapeFunscripts bool `json:"scrapeFunscripts":`
}
type RequestSaveOptionsDLNA struct {
	Enabled      bool     `json:"enabled"`
	ServiceName  string   `json:"name"`
	ServiceImage string   `json:"image"`
	AllowedIP    []string `json:"allowedIp"`
}

type RequestSaveOptionsDeoVR struct {
	Enabled                 bool   `json:"enabled"`
	AuthEnabled             bool   `json:"auth_enabled"`
	Username                string `json:"username"`
	Password                string `json:"password"`
	RemoteEnabled           bool   `json:"remote_enabled"`
	TrackWatchTime          bool   `json:"track_watch_time"`
	RenderHeatmaps          bool   `json:"render_heatmaps"`
	AllowFileDeletes        bool   `json:"allow_file_deletes"`
	AllowRatingUpdates      bool   `json:"allow_rating_updates"`
	AllowFavoriteUpdates    bool   `json:"allow_favorite_updates"`
	AllowHspData            bool   `json:"allow_hsp_data"`
	AllowTagUpdates         bool   `json:"allow_tag_updates"`
	AllowCuepointUpdates    bool   `json:"allow_cuepoint_updates"`
	AllowWatchlistUpdates   bool   `json:"allow_watchlist_updates"`
	MultitrackCuepoints     bool   `json:"multitrack_cuepoints"`
	VideoSortSeq            string `json:"video_sort_seq"`
	ScriptSortSeq           string `json:"script_sort_seq"`
	MultitrackCastCuepoints bool   `json:"multitrack_cast_cuepoints"`
	RetainNonHSPCuepoints   bool   `json:"retain_non_hsp_cuepoints"`
}

type RequestSaveOptionsPreviews struct {
	Enabled       bool    `json:"enabled"`
	StartTime     int     `json:"startTime"`
	SnippetLength float64 `json:"snippetLength"`
	SnippetAmount int     `json:"snippetAmount"`
	Resolution    int     `json:"resolution"`
	ExtraSnippet  bool    `json:"extraSnippet"`
}

type GetStateResponse struct {
	CurrentState config.ObjectState  `json:"currentState"`
	Config       config.ObjectConfig `json:"config"`
	Scrapers     []models.Scraper    `json:"scrapers"`
}

type GetSearchStateResponse struct {
	DocumentCount uint64 `json:"documentCount"`
	InProgress    bool   `json:"inProgress"`
}

type GetFunscriptCountResponse struct {
	Total   int64 `json:"total"`
	Updated int64 `json:"updated"`
}

type RequestSaveOptionsTaskSchedule struct {
	RescrapeEnabled      bool `json:"rescrapeEnabled"`
	RescrapeHourInterval int  `json:"rescrapeHourInterval"`
	RescrapeUseRange     bool `json:"rescrapeUseRange"`
	RescrapeMinuteStart  int  `json:"rescrapeMinuteStart"`
	RescrapeHourStart    int  `json:"rescrapeHourStart"`
	RescrapeHourEnd      int  `json:"rescrapeHourEnd"`
	RescrapeStartDelay   int  `json:"rescrapeStartDelay"`
	RescanEnabled        bool `json:"rescanEnabled"`
	RescanHourInterval   int  `json:"rescanHourInterval"`
	RescanUseRange       bool `json:"rescanUseRange"`
	RescanMinuteStart    int  `json:"rescanMinuteStart"`
	RescanHourStart      int  `json:"rescanHourStart"`
	RescanHourEnd        int  `json:"rescanHourEnd"`
	RescanStartDelay     int  `json:"rescanStartDelay"`
	PreviewEnabled       bool `json:"previewEnabled"`
	PreviewHourInterval  int  `json:"previewHourInterval"`
	PreviewUseRange      bool `json:"previewUseRange"`
	PreviewMinuteStart   int  `json:"previewMinuteStart"`
	PreviewHourStart     int  `json:"previewHourStart"`
	PreviewHourEnd       int  `json:"previewHourEnd"`
	PreviewStartDelay    int  `json:"previewStartDelay"`

	ActorRescrapeEnabled      bool `json:"actorRescrapeEnabled"`
	ActorRescrapeHourInterval int  `json:"actorRescrapeHourInterval"`
	ActorRescrapeUseRange     bool `json:"actorRescrapeUseRange"`
	ActorRescrapeMinuteStart  int  `json:"actorRescrapeMinuteStart"`
	ActorRescrapeHourStart    int  `json:"actorRescrapeHourStart"`
	ActorRescrapeHourEnd      int  `json:"actorRescrapeHourEnd"`
	ActorRescrapeStartDelay   int  `json:"actorRescrapeStartDelay"`

	StashdbRescrapeEnabled      bool `json:"stashdbRescrapeEnabled"`
	StashdbRescrapeHourInterval int  `json:"stashdbRescrapeHourInterval"`
	StashdbRescrapeUseRange     bool `json:"stashdbRescrapeUseRange"`
	StashdbRescrapeMinuteStart  int  `json:"stashdbRescrapeMinuteStart"`
	StashdbRescrapeHourStart    int  `json:"stashdbRescrapeHourStart"`
	StashdbRescrapeHourEnd      int  `json:"stashdbRescrapeHourEnd"`
	StashdbRescrapeStartDelay   int  `json:"stashdbRescrapeStartDelay"`
}

type RequestCuepointsResponse struct {
	Positions []string `json:"positions"`
	Actions   []string `json:"actions"`
}
type RequestSCustomSiteCreate struct {
	Url     string `json:"scraperUrl"`
	Name    string `json:"scraperName"`
	Avatar  string `json:"scraperAvatar"`
	Company string `json:"scraperCompany"`
}

type ConfigResource struct{}

func (i ConfigResource) WebService() *restful.WebService {
	tags := []string{"Options"}

	ws := new(restful.WebService)

	ws.Path("/api/options").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/version-check").To(i.versionCheck).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/state").To(i.getState).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/state/search").To(i.getSearchState).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Sites" section endpoints
	ws.Route(ws.GET("/sites").To(i.listSites).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.PUT("/sites/{site}").To(i.toggleSite).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.PUT("/sites/subscribed/{site}").To(i.toggleSubscribed).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scraper/force-site-update").To(i.forceSiteUpdate).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scraper/delete-scenes").To(i.deleteScenes).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Storage" section endpoints
	ws.Route(ws.GET("/storage").To(i.listStorage).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/storage").To(i.addStorage).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/storage/{storage-id}").To(i.removeStorage).
		Param(ws.PathParameter("storage-id", "Storage ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "DLNA" section endpoints
	ws.Route(ws.PUT("/interface/dlna").To(i.saveOptionsDLNA).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Web UI" section endpoints
	ws.Route(ws.PUT("/interface/deovr").To(i.saveOptionsDeoVR).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Web UI" section endpoints
	ws.Route(ws.PUT("/interface/web").To(i.saveOptionsWeb).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Web Advanced UI options" section endpoints
	ws.Route(ws.PUT("/interface/advanced").To(i.saveOptionsAdvanced).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Web Advanced UI options" section endpoints
	ws.Route(ws.PUT("/funscripts").To(i.saveOptionsFunscripts).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Cache" section endpoints
	ws.Route(ws.DELETE("/cache/reset/{cache}").To(i.resetCache).
		Param(ws.PathParameter("cache", "Cache to reset - possible choices are `images`, `previews`, and `searchIndex`").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Previews" section endpoints
	ws.Route(ws.PUT("/previews").To(i.saveOptionsPreviews).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/previews/test").To(i.generateTestPreview).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Funscripts" section endpoints
	ws.Route(ws.GET("/funscripts/count").To(i.getFunscriptsCount).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/task-schedule").To(i.saveOptionsTaskSchedule).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Cuepoints section endpoints"
	ws.Route(ws.GET("/cuepoints").To(i.getDefaultCuepoints).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Cuepoints section endpoints"
	ws.Route(ws.PUT("/custom-sites/create").To(i.createCustomSite).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i ConfigResource) versionCheck(req *restful.Request, resp *restful.Response) {
	out := VersionCheckResponse{LatestVersion: common.CurrentVersion, CurrentVersion: common.CurrentVersion, UpdateNotify: false}

	if config.Config.Web.UpdateCheck && common.CurrentVersion != "CURRENT" {
		r, err := resty.New().R().
			SetHeader("User-Agent", "XBVR/"+common.CurrentVersion).
			SetHeader("Accept", "application/vnd.github.v3+json").
			Get("https://api.github.com/repos/xbapps/xbvr/releases/latest")
		if err != nil || r.StatusCode() != 200 {
			resp.WriteHeaderAndEntity(http.StatusOK, out)
			return
		}

		out.LatestVersion = gjson.Get(r.String(), "tag_name").String()

		// Decide if UI notification is needed
		if version.Compare(common.CurrentVersion, out.LatestVersion, "<") {
			out.UpdateNotify = true
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) listSites(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()
	i.listSitesWithDB(req, resp, db)
}

func (i ConfigResource) toggleSite(req *restful.Request, resp *restful.Response) {
	i.toggleSiteField(req, resp, "IsEnabled")
}

func (i ConfigResource) toggleSubscribed(req *restful.Request, resp *restful.Response) {
	i.toggleSiteField(req, resp, "Subscribed")
}

func (i ConfigResource) toggleSiteField(req *restful.Request, resp *restful.Response, field string) {
	db, _ := models.GetDB()
	defer db.Close()

	id := req.PathParameter("site")
	if id == "" {
		return
	}

	var site models.Site
	err := site.GetIfExist(id)
	if err != nil {
		log.Error(err)
		return
	}
	switch field {
	case "IsEnabled":
		site.IsEnabled = !site.IsEnabled
	case "Subscribed":
		site.Subscribed = !site.Subscribed
		log.Infof("Toggling %s %v", id, site.Subscribed)
		db.Model(&models.Scene{}).Where("scraper_id = ?", site.ID).Update("is_subscribed", site.Subscribed)
	}
	site.Save()

	i.listSitesWithDB(req, resp, db)
}

func (i ConfigResource) listSitesWithDB(req *restful.Request, resp *restful.Response, db *gorm.DB) {
	var sites []models.Site
	switch db.Dialect().GetName() {
	case "mysql":
		db.Order("name asc").Find(&sites)
	case "sqlite3":
		db.Order("name COLLATE NOCASE asc").Find(&sites)
	}

	scrapers := models.GetScrapers()
	for idx, site := range sites {
		for _, scraper := range scrapers {
			if site.ID == scraper.ID {
				sites[idx].HasScraper = true
			}
		}
	}
	resp.WriteHeaderAndEntity(http.StatusOK, sites)
}

func (i ConfigResource) saveOptionsWeb(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsWeb
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	config.Config.Web.TagSort = r.TagSort
	config.Config.Web.SceneHidden = r.SceneHidden
	config.Config.Web.SceneWatchlist = r.SceneWatchlist
	config.Config.Web.SceneFavourite = r.SceneFavourite
	config.Config.Web.SceneWishlist = r.SceneWishlist
	config.Config.Web.SceneWatched = r.SceneWatched
	config.Config.Web.SceneEdit = r.SceneEdit
	config.Config.Web.SceneDuration = r.SceneDuration
	config.Config.Web.SceneCuepoint = r.SceneCuepoint
	config.Config.Web.ShowHspFile = r.ShowHspFile
	config.Config.Web.ShowSubtitlesFile = r.ShowSubtitlesFile
	config.Config.Web.SceneTrailerlist = r.SceneTrailerlist
	config.Config.Web.ShowScriptHeatmap = r.ShowScriptHeatmap
	config.Config.Web.UpdateCheck = r.UpdateCheck
	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) saveOptionsAdvanced(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsAdvanced
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	config.Config.Advanced.ShowInternalSceneId = r.ShowInternalSceneId
	config.Config.Advanced.ShowHSPApiLink = r.ShowHSPApiLink
	config.Config.Advanced.ShowSceneSearchField = r.ShowSceneSearchField
	config.Config.Advanced.StashApiKey = r.StashApiKey
	config.Config.Advanced.ScrapeActorAfterScene = r.ScrapeActorAfterScene
	config.Config.Advanced.UseImperialEntry = r.UseImperialEntry
	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}
func (i ConfigResource) saveOptionsFunscripts(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsFunscripts
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	config.Config.Funscripts.ScrapeFunscripts = r.ScrapeFunscripts
	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) saveOptionsDeoVR(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsDeoVR
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	config.Config.Interfaces.DeoVR.Enabled = r.Enabled
	config.Config.Interfaces.DeoVR.AuthEnabled = r.AuthEnabled
	config.Config.Interfaces.DeoVR.RenderHeatmaps = r.RenderHeatmaps
	config.Config.Interfaces.DeoVR.RemoteEnabled = r.RemoteEnabled
	config.Config.Interfaces.DeoVR.TrackWatchTime = r.TrackWatchTime
	config.Config.Interfaces.DeoVR.Username = r.Username
	config.Config.Interfaces.Heresphere.AllowFileDeletes = r.AllowFileDeletes
	config.Config.Interfaces.Heresphere.AllowRatingUpdates = r.AllowRatingUpdates
	config.Config.Interfaces.Heresphere.AllowFavoriteUpdates = r.AllowFavoriteUpdates
	config.Config.Interfaces.Heresphere.AllowHspData = r.AllowHspData
	config.Config.Interfaces.Heresphere.AllowTagUpdates = r.AllowTagUpdates
	config.Config.Interfaces.Heresphere.AllowCuepointUpdates = r.AllowCuepointUpdates
	config.Config.Interfaces.Heresphere.AllowWatchlistUpdates = r.AllowWatchlistUpdates
	config.Config.Interfaces.Heresphere.MultitrackCuepoints = r.MultitrackCuepoints
	config.Config.Interfaces.Players.VideoSortSeq = r.VideoSortSeq
	config.Config.Interfaces.Players.ScriptSortSeq = r.ScriptSortSeq
	config.Config.Interfaces.Heresphere.MultitrackCastCuepoints = r.MultitrackCastCuepoints
	config.Config.Interfaces.Heresphere.RetainNonHSPCuepoints = r.RetainNonHSPCuepoints
	if r.Password != config.Config.Interfaces.DeoVR.Password && r.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
		config.Config.Interfaces.DeoVR.Password = string(hash)
	}
	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) listStorage(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var vol []models.Volume
	db.Raw(`select id, path, last_scan,is_available, is_enabled, type,
       	(select count(*) from files where files.volume_id = volumes.id) as file_count,
		(select count(*) from files where files.volume_id = volumes.id and files.scene_id = 0) as unmatched_count,
       	(select sum(files.size) from files where files.volume_id = volumes.id) as total_size
		from volumes order by last_scan desc;`).Scan(&vol)

	resp.WriteHeaderAndEntity(http.StatusOK, vol)
}

func (i ConfigResource) addStorage(req *restful.Request, resp *restful.Response) {
	tlog := log.WithField("task", "rescan")

	var r NewVolumeRequest
	err := req.ReadEntity(&r)
	if err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	switch r.Type {
	case "local":
		if fi, err := os.Stat(r.Path); os.IsNotExist(err) || !fi.IsDir() {
			tlog.Error("Path does not exist or is not a directory")
			APIError(req, resp, 400, errors.New("Path does not exist or is not a directory"))
			return
		}

		path, _ := filepath.Abs(r.Path)

		var vol []models.Volume
		db.Where(&models.Volume{Path: path}).Find(&vol)

		if len(vol) > 0 {
			tlog.Error("Folder already exists")
			APIError(req, resp, 400, errors.New("Folder already exists"))
			return
		}

		nv := models.Volume{Path: path, IsEnabled: true, IsAvailable: true, Type: r.Type}
		nv.Save()

		tlog.Info("Added new storage folder ", path)

	case "putio":
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: r.Token})
		oauthClient := oauth2.NewClient(context.Background(), tokenSource)
		client := putio.NewClient(oauthClient)

		acct, err := client.Account.Info(context.Background())
		if err != nil {
			tlog.Error("Can't verify token")
			APIError(req, resp, 400, errors.New("Can't verify token"))
			return
		}

		var vol []models.Volume
		db.Where(&models.Volume{Metadata: r.Token}).Find(&vol)

		if len(vol) > 0 {
			tlog.Error("Cloud storage already exists")
			APIError(req, resp, 400, errors.New("Cloud storage already exists"))
			return
		}

		nv := models.Volume{Path: "Put.io (" + acct.Username + ")", IsEnabled: true, IsAvailable: true, Metadata: r.Token, Type: r.Type}
		nv.Save()

		tlog.Info("Added new cloud storage ", nv.Path)
	}

	// Inform UI about state change
	common.PublishWS("state.change.optionsStorage", nil)

	resp.WriteHeader(http.StatusOK)
}

func (i ConfigResource) removeStorage(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("storage-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	vol := models.Volume{}
	err = db.First(&vol, id).Error

	if err == gorm.ErrRecordNotFound {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	db.Where("volume_id = ?", id).Delete(models.File{})
	db.Delete(&vol)

	// Inform UI about state change
	common.PublishWS("state.change.optionsStorage", nil)

	tasks.RescanVolumes(-1)
	tasks.RefreshSceneStatuses()

	log.WithField("task", "rescan").Info("Removed storage", vol.Path)

	resp.WriteHeader(http.StatusOK)
}

func (i ConfigResource) forceSiteUpdate(req *restful.Request, resp *restful.Response) {
	var r struct {
		ScraperId string `json:"scraper_id"`
	}

	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	db.Model(&models.Scene{}).Where("scraper_id = ?", r.ScraperId).Update("needs_update", true)
}

func (i ConfigResource) deleteScenes(req *restful.Request, resp *restful.Response) {
	var r struct {
		ScraperId string `json:"scraper_id"`
	}

	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Where("scraper_id = ?", r.ScraperId).Find(&scenes)

	for _, obj := range scenes {
		files, _ := obj.GetFiles()
		for _, file := range files {
			file.SceneID = 0
			file.Save()
		}
	}

	db.Where("scraper_id = ?", r.ScraperId).Delete(&models.Scene{})
}

func (i ConfigResource) getState(req *restful.Request, resp *restful.Response) {
	var out GetStateResponse

	tasks.UpdateState()

	out.Config = config.Config
	out.CurrentState = config.State
	out.Scrapers = models.GetScrapers()

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) getSearchState(req *restful.Request, resp *restful.Response) {
	var out GetSearchStateResponse

	out.InProgress = models.CheckLock("index")
	out.DocumentCount = 0
	if !out.InProgress { // don't open if in progress, bleve open will hang
		idx, err := tasks.NewIndex("scenes")
		if err == nil {
			defer idx.Bleve.Close()
			out.DocumentCount, _ = idx.Bleve.DocCount()
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) resetCache(req *restful.Request, resp *restful.Response) {
	cache := req.PathParameter("cache")

	if cache == "images" {
		os.RemoveAll(common.ImgDir)
		os.MkdirAll(common.ImgDir, os.ModePerm)
		config.State.CacheSize.Images = 0
	}

	if cache == "searchIndex" {
		os.RemoveAll(common.IndexDirV2)
		os.MkdirAll(common.IndexDirV2, os.ModePerm)
		config.State.CacheSize.SearchIndex = 0
	}

	if cache == "previews" {
		db, _ := models.GetDB()
		db.Model(&models.Scene{}).Where("has_video_preview = ?", true).Update("has_video_preview", false)
		db.Close()

		os.RemoveAll(common.VideoPreviewDir)
		os.MkdirAll(common.VideoPreviewDir, os.ModePerm)
		config.State.CacheSize.Previews = 0
	}

	config.SaveState()

	resp.WriteHeader(http.StatusOK)
}

func (i ConfigResource) saveOptionsDLNA(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsDLNA
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	config.Config.Interfaces.DLNA.Enabled = r.Enabled
	config.Config.Interfaces.DLNA.ServiceName = r.ServiceName
	config.Config.Interfaces.DLNA.ServiceImage = r.ServiceImage
	config.Config.Interfaces.DLNA.AllowedIP = r.AllowedIP
	config.SaveConfig()

	if tasks.IsDMSStarted() {
		tasks.StopDMS()
		time.Sleep(1 * time.Second)
	}

	if r.Enabled {
		tasks.StartDMS()
	}

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) saveOptionsPreviews(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsPreviews
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	config.Config.Library.Preview.Resolution = r.Resolution
	config.Config.Library.Preview.SnippetAmount = r.SnippetAmount
	config.Config.Library.Preview.StartTime = r.StartTime
	config.Config.Library.Preview.SnippetLength = r.SnippetLength
	config.Config.Library.Preview.ExtraSnippet = r.ExtraSnippet
	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) generateTestPreview(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsPreviews
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	// Get first available scene for test preview
	var scene models.Scene
	db, _ := models.GetDB()
	db.Model(&models.Scene{}).Where("is_available = ?", true).Order("release_date desc").First(&scene)
	db.Close()

	files, err := scene.GetFiles()
	if err != nil {
		log.Error(err)
		return
	}

	// Generate hash for given parameters
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("test-%v-%v-%v-%v-%v-%v", scene.SceneID, r.StartTime, r.SnippetLength, r.SnippetAmount, r.Resolution, r.ExtraSnippet)))

	previewFn := fmt.Sprintf("test%x", hash.Sum(nil))
	destFile := filepath.Join(common.VideoPreviewDir, previewFn+".mp4")

	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		// Preview file does not exist, generate it
		go func() {
			tasks.RenderPreview(
				files[0].GetPath(),
				destFile,
				files[0].VideoProjection,
				r.StartTime,
				r.SnippetLength,
				r.SnippetAmount,
				r.Resolution,
				r.ExtraSnippet,
			)

			common.PublishWS("options.previews.previewReady", map[string]interface{}{"previewFn": previewFn})
		}()

		resp.WriteHeader(http.StatusOK)
		return
	}

	common.PublishWS("options.previews.previewReady", map[string]interface{}{"previewFn": previewFn})
	resp.WriteHeader(http.StatusOK)
}

func (i ConfigResource) getFunscriptsCount(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r GetFunscriptCountResponse

	var scenes []models.Scene
	db.Model(&models.Scene{}).Preload("Files", func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ?", "script").Order("is_selected_script DESC, created_time DESC")
	}).Where("is_scripted = ?", true).Find(&scenes)

	for _, scene := range scenes {
		r.Total++
		if len(scene.Files) > 0 && !scene.Files[0].IsExported {
			r.Updated++
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) saveOptionsTaskSchedule(req *restful.Request, resp *restful.Response) {
	var r RequestSaveOptionsTaskSchedule
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.RescrapeHourEnd > 23 {
		r.RescrapeHourEnd -= 24
	}
	if r.RescanHourEnd > 23 {
		r.RescanHourEnd -= 24
	}
	if r.PreviewHourEnd > 23 {
		r.PreviewHourEnd -= 24
	}

	config.Config.Cron.RescrapeSchedule.Enabled = r.RescrapeEnabled
	config.Config.Cron.RescrapeSchedule.HourInterval = r.RescrapeHourInterval
	config.Config.Cron.RescrapeSchedule.UseRange = r.RescrapeUseRange
	config.Config.Cron.RescrapeSchedule.MinuteStart = r.RescrapeMinuteStart
	config.Config.Cron.RescrapeSchedule.HourStart = r.RescrapeHourStart
	config.Config.Cron.RescrapeSchedule.HourEnd = r.RescrapeHourEnd
	config.Config.Cron.RescrapeSchedule.RunAtStartDelay = r.RescrapeStartDelay

	config.Config.Cron.RescanSchedule.Enabled = r.RescanEnabled
	config.Config.Cron.RescanSchedule.HourInterval = r.RescanHourInterval
	config.Config.Cron.RescanSchedule.UseRange = r.RescanUseRange
	config.Config.Cron.RescanSchedule.MinuteStart = r.RescanMinuteStart
	config.Config.Cron.RescanSchedule.HourStart = r.RescanHourStart
	config.Config.Cron.RescanSchedule.HourEnd = r.RescanHourEnd
	config.Config.Cron.RescanSchedule.RunAtStartDelay = r.RescanStartDelay

	config.Config.Cron.PreviewSchedule.Enabled = r.PreviewEnabled
	config.Config.Cron.PreviewSchedule.HourInterval = r.PreviewHourInterval
	config.Config.Cron.PreviewSchedule.UseRange = r.PreviewUseRange
	config.Config.Cron.PreviewSchedule.MinuteStart = r.PreviewMinuteStart
	config.Config.Cron.PreviewSchedule.HourStart = r.PreviewHourStart
	config.Config.Cron.PreviewSchedule.HourEnd = r.PreviewHourEnd
	config.Config.Cron.PreviewSchedule.RunAtStartDelay = r.PreviewStartDelay

	config.Config.Cron.ActorRescrapeSchedule.Enabled = r.ActorRescrapeEnabled
	config.Config.Cron.ActorRescrapeSchedule.HourInterval = r.ActorRescrapeHourInterval
	config.Config.Cron.ActorRescrapeSchedule.UseRange = r.ActorRescrapeUseRange
	config.Config.Cron.ActorRescrapeSchedule.MinuteStart = r.ActorRescrapeMinuteStart
	config.Config.Cron.ActorRescrapeSchedule.HourStart = r.ActorRescrapeHourStart
	config.Config.Cron.ActorRescrapeSchedule.HourEnd = r.ActorRescrapeHourEnd
	config.Config.Cron.ActorRescrapeSchedule.RunAtStartDelay = r.ActorRescrapeStartDelay

	config.Config.Cron.StashdbRescrapeSchedule.Enabled = r.StashdbRescrapeEnabled
	config.Config.Cron.StashdbRescrapeSchedule.HourInterval = r.StashdbRescrapeHourInterval
	config.Config.Cron.StashdbRescrapeSchedule.UseRange = r.StashdbRescrapeUseRange
	config.Config.Cron.StashdbRescrapeSchedule.MinuteStart = r.StashdbRescrapeMinuteStart
	config.Config.Cron.StashdbRescrapeSchedule.HourStart = r.StashdbRescrapeHourStart
	config.Config.Cron.StashdbRescrapeSchedule.HourEnd = r.StashdbRescrapeHourEnd
	config.Config.Cron.StashdbRescrapeSchedule.RunAtStartDelay = r.StashdbRescrapeStartDelay

	config.SaveConfig()

	resp.WriteHeaderAndEntity(http.StatusOK, r)
}

func (i ConfigResource) getDefaultCuepoints(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var kv models.KV
	kv.Key = "cuepoints"
	db.Find(&kv)

	var cp RequestCuepointsResponse
	json.Unmarshal([]byte(kv.Value), &cp)
	resp.WriteHeaderAndEntity(http.StatusOK, &cp)
}

func (i ConfigResource) createCustomSite(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestSCustomSiteCreate
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}
	if r.Url == "" || r.Name == "" {
		return
	}
	r.Url = strings.TrimSpace(r.Url)
	r.Name = strings.TrimSpace(r.Name)
	r.Company = strings.TrimSpace(r.Company)
	r.Avatar = strings.TrimSpace(r.Avatar)
	if r.Company == "" {
		r.Company = r.Name
	}

	var scraperConfig config.ScraperList
	scraperConfig.Load()

	re := regexp.MustCompile(`^(https?://)?(www\.)?([^./]+)\.`)
	match := re.FindStringSubmatch(r.Url)
	if len(match) < 3 {
		return
	}

	r.Url = strings.TrimSuffix(r.Url, "/")

	scrapers := make(map[string][]config.ScraperConfig)
	scrapers["povr"] = scraperConfig.CustomScrapers.PovrScrapers
	scrapers["slr"] = scraperConfig.CustomScrapers.SlrScrapers
	scrapers["vrphub"] = scraperConfig.CustomScrapers.VrphubScrapers
	scrapers["vrporn"] = scraperConfig.CustomScrapers.VrpornScrapers

	exists := false
	for key, group := range scrapers {
		for idx, site := range group {
			if site.URL == r.Url {
				exists = true
				scrapers[key][idx].Name = r.Name
				scrapers[key][idx].Company = r.Company
				scrapers[key][idx].AvatarUrl = r.Avatar
			}
		}
	}

	if !exists {
		scraper := config.ScraperConfig{URL: r.Url, Name: r.Name, Company: r.Company, AvatarUrl: r.Avatar}
		switch match[3] {
		case "povr":
			scrapers["povr"] = append(scrapers["povrr"], scraper)
		case "sexlikereal":
			scrapers["slr"] = append(scrapers["slr"], scraper)
		case "vrphub":
			scrapers["vrphub"] = append(scrapers["vrphub"], scraper)
		case "vrporn":
			scrapers["vrporn"] = append(scrapers["vrporn"], scraper)
		}
	}
	scraperConfig.CustomScrapers.PovrScrapers = scrapers["povr"]
	scraperConfig.CustomScrapers.SlrScrapers = scrapers["slr"]
	scraperConfig.CustomScrapers.VrphubScrapers = scrapers["vrphub"]
	scraperConfig.CustomScrapers.VrpornScrapers = scrapers["vrporn"]
	fName := filepath.Join(common.AppDir, "scrapers.json")
	list, _ := json.MarshalIndent(scraperConfig, "", "  ")
	ioutil.WriteFile(fName, list, 0644)

	resp.WriteHeader(http.StatusOK)
}
