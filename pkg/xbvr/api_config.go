package xbvr

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
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

	ws.Route(ws.GET("/volume").To(i.listVolume).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/volume").To(i.addVolume).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/volume").To(i.deleteVolume).
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

func (i ConfigResource) listVolume(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Raw(`select path, last_scan,is_available, is_enabled,
       	(select count(*) from files where files.path like volumes.path || "%") as file_count,
		(select count(*) from files where files.path like volumes.path || "%" and files.scene_id = 0) as unmatched_count,
       	(select sum(files.size) from files where files.path like volumes.path || "%") as total_size
		from volumes order by last_scan desc;`).Scan(&vol)

	resp.WriteHeaderAndEntity(http.StatusOK, vol)
}

func (i ConfigResource) addVolume(req *restful.Request, resp *restful.Response) {
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

	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Where(&Volume{Path: path}).Find(&vol)

	if len(vol) > 0 {
		tlog.Error("Folder already exists")
		APIError(req, resp, 400, errors.New("Folder already exists"))
		return
	}

	nv := Volume{Path: path, IsEnabled: true, IsAvailable: true}
	nv.Save()

	tlog.Info("Added new folder", path)
}

func (i ConfigResource) deleteVolume(req *restful.Request, resp *restful.Response) {

}
