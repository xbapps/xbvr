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
	"github.com/putdotio/go-putio/putio"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
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

	ws.Route(ws.GET("/storage").To(i.listStorage).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/storage").To(i.addStorage).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/storage/{storage-id}").To(i.removeStorage).
		Param(ws.PathParameter("storage-id", "Storage ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/scraper/force-site-update").To(i.forceSiteUpdate).
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
	publisher, err := client.ConnectNet(context.Background(), "ws://"+wsAddr+"/ws", client.Config{Realm: "default"})
	if err == nil {
		publisher.Publish("state.change.optionsFolders", nil, nil, nil)
		publisher.Close()
	}

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
	publisher, err := client.ConnectNet(context.Background(), "ws://"+wsAddr+"/ws", client.Config{Realm: "default"})
	if err == nil {
		publisher.Publish("state.change.optionsFolders", nil, nil, nil)
		publisher.Close()
	}

	RescanVolumes()

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
