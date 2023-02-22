package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
)

type DMSResource struct{}

var (
	lastSessionID      uint
	lastSessionSceneID uint
	lastSessionStart   time.Time
	lastSessionEnd     time.Time
)

func (i DMSResource) WebService() *restful.WebService {
	tags := []string{"DMS"}

	ws := new(restful.WebService)

	ws.Path("/api/dms").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/file/{file-id}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/file/{file-id}/{var:*}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/heatmap/{file-id}").To(i.getHeatmap).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/preview/{scene-id}").To(i.getPreview).
		Param(ws.PathParameter("scene-id", "Scene ID")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i DMSResource) getPreview(req *restful.Request, resp *restful.Response) {
	sceneID := req.PathParameter("scene-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", sceneID)))
}

func (i DMSResource) getHeatmap(req *restful.Request, resp *restful.Response) {
	fileID := req.PathParameter("file-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%v.png", fileID)))
}

func (i DMSResource) getFile(req *restful.Request, resp *restful.Response) {
	doNotTrack := req.QueryParameter("dnt")
	id, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if scene exist
	db, _ := models.GetDB()
	defer db.Close()

	f := models.File{}
	err = db.Preload("Volume").First(&f, id).Error

	switch f.Volume.Type {
	case "local":
		// Track current session
		setDeoPlayerHost(req)
		session.TrackSessionFromFile(f, doNotTrack)

		if err == gorm.ErrRecordNotFound {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		ctx := req.Request.Context()
		http.ServeFile(resp.ResponseWriter, req.Request, f.GetPath())
		select {
		case <-ctx.Done():
			session.FinishTrackingFromFile(doNotTrack)
			return
		default:
		}
	case "putio":
		id, err := strconv.ParseInt(f.Path, 10, 64)
		if err != nil {
			return
		}
		client := f.Volume.GetPutIOClient()
		url, err := client.Files.URL(context.Background(), id, false)
		if err != nil {
			return
		}
		http.Redirect(resp.ResponseWriter, req.Request, url, 302)
	}
}
