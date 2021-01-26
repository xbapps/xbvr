package api

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/putdotio/go-putio/putio"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/assets"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
	"golang.org/x/oauth2"
	"gopkg.in/resty.v1"
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
	TagSort      string   `json:"tagSort"`
	SceneEdit    bool     `json:"sceneEdit"`
}

type RequestSaveOptionsDLNA struct {
	Enabled      bool     `json:"enabled"`
	ServiceName  string   `json:"name"`
	ServiceImage string   `json:"image"`
	AllowedIP    []string `json:"allowedIp"`
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
	CurrentState struct {
		Web struct {
			TagSort   string  `json:"tagSort"`
			SceneEdit bool    `json:"sceneEdit"`
		} `json:"web"`
		DLNA struct {
			Running  bool     `json:"running"`
			Images   []string `json:"images"`
			RecentIP []string `json:"recentIp"`
		} `json:"dlna"`
		CacheSize struct {
			Images      int64 `json:"images"`
			Previews    int64 `json:"previews"`
			SearchIndex int64 `json:"searchIndex"`
		} `json:"cacheSize"`
	} `json:"currentState"`
	Config config.Object `json:"config"`
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

	// "Sites" section endpoints
	ws.Route(ws.GET("/sites").To(i.listSites).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.PUT("/sites/{site}").To(i.toggleSite).
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
	ws.Route(ws.PUT("/interface/web").To(i.saveOptionsWeb).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Cache" section endpoints
	ws.Route(ws.DELETE("/cache/reset/{cache}").To(i.resetCache).
		Param(ws.PathParameter("cache", "Cache to reset").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	// "Previews" section endpoints
	ws.Route(ws.PUT("/previews").To(i.saveOptionsPreviews).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/previews/test").To(i.generateTestPreview).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i ConfigResource) versionCheck(req *restful.Request, resp *restful.Response) {
	out := VersionCheckResponse{LatestVersion: common.CurrentVersion, CurrentVersion: common.CurrentVersion, UpdateNotify: false}

	if common.CurrentVersion != "CURRENT" {
		r, err := resty.R().
			SetHeader("User-Agent", "XBVR/"+common.CurrentVersion).
			Get("https://updates.xbvr.app/latest.json")
		if err != nil || r.StatusCode() != 200 {
			resp.WriteHeaderAndEntity(http.StatusOK, out)
			return
		}

		out.LatestVersion = gjson.Get(r.String(), "latestVersion").String()

		// Decide if UI notification is needed
		if out.LatestVersion != common.CurrentVersion {
			out.UpdateNotify = true
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) listSites(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var sites []models.Site
	db.Order("name asc").Find(&sites)

	resp.WriteHeaderAndEntity(http.StatusOK, sites)
}

func (i ConfigResource) toggleSite(req *restful.Request, resp *restful.Response) {
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
	site.IsEnabled = !site.IsEnabled
	site.Save()

	var sites []models.Site
	db.Order("name asc").Find(&sites)
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
	config.Config.Web.SceneEdit = r.SceneEdit
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

	tasks.RescanVolumes()

	log.WithField("task", "rescan").Info("Removed storage", vol.Path)

	resp.WriteHeader(http.StatusOK)
}

func (i ConfigResource) forceSiteUpdate(req *restful.Request, resp *restful.Response) {
	var r struct {
		SiteName string `json:"site_name"`
	}

	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	db.Model(&models.Scene{}).Where("site = ?", r.SiteName).Update("needs_update", true)
}

func (i ConfigResource) deleteScenes(req *restful.Request, resp *restful.Response) {
	var r struct {
		SiteName string `json:"site_name"`
	}

	if err := req.ReadEntity(&r); err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Where("site = ?", r.SiteName).Find(&scenes)

	for _, obj := range scenes {
		files, _ := obj.GetFiles()
		for _, file := range files {
			file.SceneID = 0
			file.Save()
		}
	}

	db.Where("site = ?", r.SiteName).Delete(&models.Scene{})
}

func (i ConfigResource) getState(req *restful.Request, resp *restful.Response) {
	var out GetStateResponse
	out.Config = config.Config

	// Preferences
	out.CurrentState.Web.TagSort = config.Config.Web.TagSort
	out.CurrentState.Web.SceneEdit = config.Config.Web.SceneEdit

	// DLNA
	out.CurrentState.DLNA.Running = tasks.IsDMSStarted()
	out.CurrentState.DLNA.RecentIP = config.RecentIPAddresses
	dlnaImages, _ := assets.WalkDirs("dlna", false)
	for _, v := range dlnaImages {
		out.CurrentState.DLNA.Images = append(out.CurrentState.DLNA.Images, strings.Replace(strings.Split(v, "/")[1], ".png", "", -1))
	}

	// Caches
	out.CurrentState.CacheSize.Images, _ = common.DirSize(common.ImgDir)
	out.CurrentState.CacheSize.Previews, _ = common.DirSize(common.VideoPreviewDir)
	out.CurrentState.CacheSize.SearchIndex, _ = common.DirSize(common.IndexDirV2)

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) resetCache(req *restful.Request, resp *restful.Response) {
	cache := req.PathParameter("cache")

	if cache == "images" {
		os.RemoveAll(common.ImgDir)
		os.MkdirAll(common.ImgDir, os.ModePerm)
	}

	if cache == "searchIndex" {
		os.RemoveAll(common.IndexDirV2)
		os.MkdirAll(common.IndexDirV2, os.ModePerm)
	}

	if cache == "previews" {
		db, _ := models.GetDB()
		db.Model(&models.Scene{}).Where("has_video_preview = ?", true).Update("has_video_preview", false)
		db.Close()

		os.RemoveAll(common.VideoPreviewDir)
		os.MkdirAll(common.VideoPreviewDir, os.ModePerm)
	}

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
