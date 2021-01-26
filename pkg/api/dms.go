package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
	"github.com/posthog/posthog-go"
	"github.com/xbapps/xbvr/pkg/analytics"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
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

	ws.Route(ws.GET("/preview/{scene-id}").To(i.getPreview).
		Param(ws.PathParameter("scene-id", "Scene ID")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i DMSResource) getPreview(req *restful.Request, resp *restful.Response) {
	sceneID := req.PathParameter("scene-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", sceneID)))
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
		if f.SceneID != 0 && doNotTrack == "" {
			if lastSessionSceneID != f.SceneID {
				if lastSessionID != 0 {
					watchSessionFlush()
				}

				lastSessionSceneID = f.SceneID
				lastSessionStart = time.Now()
				newWatchSession()
			}
		}

		if err == gorm.ErrRecordNotFound {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		ctx := req.Request.Context()
		http.ServeFile(resp.ResponseWriter, req.Request, f.GetPath())
		select {
		case <-ctx.Done():
			lastSessionEnd = time.Now()
			if doNotTrack == "" {
				watchSessionFlush()
			}
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

func newWatchSession() {
	obj := models.History{SceneID: lastSessionSceneID, TimeStart: lastSessionStart}
	obj.Save()

	var scene models.Scene
	err := scene.GetIfExistByPK(lastSessionSceneID)
	if err == nil {
		scene.LastOpened = time.Now()
		scene.Save()
	}

	analytics.Event("watchsession-new", posthog.NewProperties().Set("scene-id", scene.SceneID))

	lastSessionID = obj.ID
}

func watchSessionFlush() {
	var obj models.History
	err := obj.GetIfExist(lastSessionID)
	if err == nil {
		obj.TimeEnd = lastSessionEnd
		obj.Duration = time.Since(lastSessionStart).Seconds()
		obj.Save()

		var scene models.Scene
		err := scene.GetIfExistByPK(lastSessionSceneID)
		if err == nil {
			if !scene.IsWatched {
				scene.IsWatched = true
				scene.Save()
			}
		}

		log.Infof("Session #%v duration for scene #%v is %v", lastSessionID, lastSessionSceneID, time.Since(lastSessionStart).Seconds())
	}
}

func CheckForDeadSession() {
	if time.Since(lastSessionEnd).Seconds() > 60 && lastSessionSceneID != 0 && lastSessionID != 0 {
		watchSessionFlush()
		lastSessionID = 0
		lastSessionSceneID = 0
	}
}
