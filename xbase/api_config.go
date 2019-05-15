package xbase

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/pkg/errors"
)

type NewVolumeRequest struct {
	Path string `json:"path"`
}

type ConfigResource struct{}

func (i ConfigResource) WebService() *restful.WebService {
	tags := []string{"Config"}

	ws := new(restful.WebService)

	ws.Path("/api/config").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/volume").To(i.listVolume).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/volume").To(i.addVolume).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/volume").To(i.deleteVolume).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i ConfigResource) listVolume(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Model(&Volume{}).Order("last_scan desc").Find(&vol)

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
		tlog.Error("Volume already exists")
		APIError(req, resp, 400, errors.New("Volume already exists"))
		return
	}

	nv := Volume{Path: path, IsEnabled: true, IsAvailable: true}
	nv.Save()

	tlog.Info("Added new volume", path)
}

func (i ConfigResource) deleteVolume(req *restful.Request, resp *restful.Response) {

}
