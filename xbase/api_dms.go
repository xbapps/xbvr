package xbase

import (
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
)

type DMSDataResponse struct {
	Sites        []string `json:"sites"`
	Actors       []string `json:"actors"`
	Tags         []string `json:"tags"`
	ReleaseGroup []string `json:"release_group"`
	Volumes      []Volume `json:"volumes"`
}

type DMSResource struct{}

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

	ws.Route(ws.GET("/file/{file-id}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i DMSResource) sceneById(req *restful.Request, resp *restful.Response) {
	sceneId := req.QueryParameter("id")

	db, _ := GetDB()
	defer db.Close()

	var scene Scene
	db.Preload("Cast").
		Preload("Tags").
		Preload("Filenames").
		Preload("Images").
		Preload("Files").
		Where(&Scene{SceneID: sceneId}).FirstOrCreate(&scene)

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i DMSResource) base(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	// Get all accessible scenes
	var scenes []Scene
	tx := db.
		Model(&scenes).
		Preload("Cast").
		Preload("Tags").
		Preload("Filenames").
		Preload("Images").
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
	var vol []Volume
	db.Where("is_available = ?", true).Find(&vol)

	resp.WriteHeaderAndEntity(http.StatusOK, DMSDataResponse{Sites: outSites, Tags: outTags, Actors: outCast, Volumes: vol, ReleaseGroup: outRelease})
}

func (i DMSResource) getFile(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if scene exist
	db, _ := GetDB()
	defer db.Close()

	f := File{}
	err = db.First(&f, id).Error

	if err == gorm.ErrRecordNotFound {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	http.ServeFile(resp.ResponseWriter, req.Request, f.GetPath())
}
