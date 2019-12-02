package xbvr

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/models"
)

type DMSDataResponse struct {
	Sites        []string        `json:"sites"`
	Actors       []string        `json:"actors"`
	Tags         []string        `json:"tags"`
	ReleaseGroup []string        `json:"release_group"`
	Volumes      []models.Volume `json:"volumes"`
}

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

	ws.Route(ws.GET("/base").To(i.base).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scene").To(i.sceneById).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/file").To(i.fileById).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/file/{file-id}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i DMSResource) sceneById(req *restful.Request, resp *restful.Response) {
	sceneId := req.QueryParameter("id")

	db, _ := models.GetDB()
	defer db.Close()

	var scene models.Scene
	db.Preload("Cast").
		Preload("Tags").
		Preload("Files").
		Where(&models.Scene{SceneID: sceneId}).First(&scene)

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i DMSResource) fileById(req *restful.Request, resp *restful.Response) {
	fileId, err := strconv.Atoi(req.QueryParameter("id"))
	if err != nil {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var file models.File
	db.Where(&models.File{ID: uint(fileId)}).First(&file)

	resp.WriteHeaderAndEntity(http.StatusOK, file)
}

func (i DMSResource) base(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	// Get all accessible scenes
	var scenes []models.Scene
	tx := db.
		Model(&scenes).
		Preload("Cast").
		Preload("Tags").
		Preload("Files")

	tx = tx.Where("is_accessible = ?", 1)

	// Available sites
	tx.Group("site").Find(&scenes)
	var outSites []string
	for i := range scenes {
		if scenes[i].Site != "" {
			outSites = append(outSites, scenes[i].Site)
		}
	}

	// Available release dates (YYYY-MM)
	tx.Select("strftime('%Y-%m', release_date) as release_date_text").
		Group("strftime('%Y-%m', release_date)").Find(&scenes)
	var outRelease []string
	for i := range scenes {
		outRelease = append(outRelease, scenes[i].ReleaseDateText)
	}

	// Available tags
	tx.Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
		Joins("left join tags on tags.id=scene_tags.tag_id").
		Group("tags.name").Select("tags.name as release_date_text").Find(&scenes)

	var outTags []string
	for i := range scenes {
		if scenes[i].ReleaseDateText != "" {
			outTags = append(outTags, scenes[i].ReleaseDateText)
		}
	}

	// Available actors
	tx.Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
		Joins("left join actors on actors.id=scene_cast.actor_id").
		Group("actors.name").Select("actors.name as release_date_text").Find(&scenes)

	var outCast []string
	for i := range scenes {
		if scenes[i].ReleaseDateText != "" {
			outCast = append(outCast, scenes[i].ReleaseDateText)
		}
	}

	// Available volumes
	var vol []models.Volume
	db.Where("is_available = ?", true).Find(&vol)

	resp.WriteHeaderAndEntity(http.StatusOK, DMSDataResponse{Sites: outSites, Tags: outTags, Actors: outCast, Volumes: vol, ReleaseGroup: outRelease})
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

func checkForDeadSession() {
	if time.Since(lastSessionEnd).Seconds() > 60 && lastSessionSceneID != 0 && lastSessionID != 0 {
		watchSessionFlush()
		lastSessionID = 0
		lastSessionSceneID = 0
	}
}
