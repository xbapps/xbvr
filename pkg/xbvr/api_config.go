package xbvr

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/blang/semver"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/gammazero/nexus/v3/client"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/resty.v1"
)

type NewVolumeRequest struct {
	Path string `json:"path"`
}

type VersionCheckResponse struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateNotify   bool   `json:"update_notify"`
}

type ConfigResource struct{}

func (i ConfigResource) WebService() *restful.WebService {
	tags := []string{"Config"}

	ws := new(restful.WebService)

	ws.Path("/api/config").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/version-check").To(i.versionCheck).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/sites").To(i.listSites).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.PUT("/sites/{site}").To(i.toggleSite).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/folder").To(i.listFolders).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/folder").To(i.addFolder).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/folder/{folder-id}").To(i.removeFolder).
		Param(ws.PathParameter("folder-id", "Folder ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i ConfigResource) versionCheck(req *restful.Request, resp *restful.Response) {
	out := VersionCheckResponse{LatestVersion: currentVersion, CurrentVersion: currentVersion, UpdateNotify: false}

	if currentVersion != "CURRENT" {
		r, err := resty.R().
			SetHeader("User-Agent", "XBVR/"+currentVersion).
			Get("https://updates.xbvr.app/latest.json")
		if err != nil || r.StatusCode() != 200 {
			resp.WriteHeaderAndEntity(http.StatusOK, out)
			return
		}

		out.LatestVersion = gjson.Get(r.String(), "latestVersion").String()

		// Decide if UI notification is needed
		sLatest := semver.MustParse(out.LatestVersion)
		sCurrent := semver.MustParse(currentVersion)
		if sLatest.GT(sCurrent) {
			out.UpdateNotify = true
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i ConfigResource) listSites(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var sites []models.Site
	db.Order("is_enabled desc").Find(&sites)

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
	db.Order("is_enabled desc").Find(&sites)
	resp.WriteHeaderAndEntity(http.StatusOK, sites)
}

func (i ConfigResource) listFolders(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var vol []models.Volume
	db.Raw(`select id, path, last_scan,is_available, is_enabled,
       	(select count(*) from files where files.path like volumes.path || "%") as file_count,
		(select count(*) from files where files.path like volumes.path || "%" and files.scene_id = 0) as unmatched_count,
       	(select sum(files.size) from files where files.path like volumes.path || "%") as total_size
		from volumes order by last_scan desc;`).Scan(&vol)

	resp.WriteHeaderAndEntity(http.StatusOK, vol)
}

func (i ConfigResource) addFolder(req *restful.Request, resp *restful.Response) {
	tlog := log.WithField("task", "rescan")

	var r NewVolumeRequest
	err := req.ReadEntity(&r)
	if err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	if fi, err := os.Stat(r.Path); os.IsNotExist(err) || !fi.IsDir() {
		tlog.Error("Path does not exist or is not a directory")
		APIError(req, resp, 400, errors.New("Path does not exist or is not a directory"))
		return
	}

	path, _ := filepath.Abs(r.Path)

	db, _ := models.GetDB()
	defer db.Close()

	var vol []models.Volume
	db.Where(&models.Volume{Path: path}).Find(&vol)

	if len(vol) > 0 {
		tlog.Error("Folder already exists")
		APIError(req, resp, 400, errors.New("Folder already exists"))
		return
	}

	nv := models.Volume{Path: path, IsEnabled: true, IsAvailable: true}
	nv.Save()

	tlog.Info("Added new folder", path)

	// Inform UI about state change
	publisher, err := client.ConnectNet(context.Background(), "ws://"+wsAddr+"/ws", client.Config{Realm: "default"})
	if err == nil {
		publisher.Publish("state.change.optionsFolders", nil, nil, nil)
		publisher.Close()
	}

	resp.WriteHeader(http.StatusOK)
}

func (i ConfigResource) removeFolder(req *restful.Request, resp *restful.Response) {
	log.Info("delete?")
	id, err := strconv.Atoi(req.PathParameter("folder-id"))
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
	publisher, err := client.ConnectNet(context.Background(), "ws://"+wsAddr+"/ws", client.Config{Realm: "default"})
	if err == nil {
		publisher.Publish("state.change.optionsFolders", nil, nil, nil)
		publisher.Close()
	}

	log.WithField("task", "rescan").Info("Removed folder", vol.Path)

	resp.WriteHeader(http.StatusOK)
}
