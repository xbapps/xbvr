package xbvr

import (
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type RequestToggleList struct {
	SceneID string `json:"scene_id"`
	List    string `json:"list"`
}

type ResponseGetScenes struct {
	Results int     `json:"results"`
	Scenes  []Scene `json:"scenes"`
}

type ResponseGetFilters struct {
	Cast          []string `json:"cast"`
	Tags          []string `json:"tags"`
	Sites         []string `json:"sites"`
	ReleaseMonths []string `json:"release_month"`
}

type SceneResource struct{}

func (i SceneResource) WebService() *restful.WebService {
	tags := []string{"Scene"}

	ws := new(restful.WebService)

	ws.Path("/api/scene").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/list").To(i.getScenes).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.POST("/toggle").To(i.toggleList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.GET("/filters/all").To(i.getFiltersAll).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetFilters{}))

	ws.Route(ws.GET("/filters/state").To(i.getFiltersForState).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetFilters{}))

	return ws
}

func (i SceneResource) getFiltersAll(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	var tags []Tag
	db.Model(&Tag{}).Order("name").Find(&tags)

	var outTags []string
	for i := range tags {
		outTags = append(outTags, tags[i].Name)
	}

	var actors []Actor
	db.Model(&Actor{}).Order("name").Find(&actors)

	var outCast []string
	for i := range actors {
		outCast = append(outCast, actors[i].Name)
	}

	var scenes []Scene
	db.Model(&Scene{}).Order("site").Group("site").Find(&scenes)

	var outSites []string
	for i := range scenes {
		if scenes[i].Site != "" {
			outSites = append(outSites, scenes[i].Site)
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetFilters{Tags: outTags, Cast: outCast, Sites: outSites})
}

func (i SceneResource) getFiltersForState(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	// Get all accessible scenes
	var scenes []Scene
	tx := db.
		Model(&scenes).
		Preload("Cast").
		Preload("Tags").
		Preload("Files")

	if req.QueryParameter("is_available") != "" {
		q_is_available, err := strconv.ParseBool(req.QueryParameter("is_available"))
		if err == nil {
			tx = tx.Where("is_available = ?", q_is_available)
		}
	}

	if req.QueryParameter("is_accessible") != "" {
		q_is_accessible, err := strconv.ParseBool(req.QueryParameter("is_accessible"))
		if err == nil {
			tx = tx.Where("is_accessible = ?", q_is_accessible)
		}
	}

	// Available sites
	tx.Group("site").Find(&scenes)
	var outSites []string
	for i := range scenes {
		if scenes[i].Site != "" {
			outSites = append(outSites, scenes[i].Site)
		}
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

	// Available release dates (YYYY-MM)
	tx.Select("strftime('%Y-%m', release_date) as release_date_text").
		Group("strftime('%Y-%m', release_date)").Find(&scenes)
	var outRelease []string
	for i := range scenes {
		outRelease = append(outRelease, scenes[i].ReleaseDateText)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetFilters{Tags: outTags, Cast: outCast, Sites: outSites, ReleaseMonths: outRelease})
}

func (i SceneResource) getScenes(req *restful.Request, resp *restful.Response) {
	var limit = 100
	var offset = 0
	var total = 0

	q_limit, err := strconv.Atoi(req.QueryParameter("limit"))
	if err == nil {
		limit = q_limit
	}

	q_offset, err := strconv.Atoi(req.QueryParameter("offset"))
	if err == nil {
		offset = q_offset
	}

	db, _ := GetDB()
	defer db.Close()

	var scenes []Scene
	tx := db.
		Model(&scenes).
		Preload("Cast").
		Preload("Tags").
		Preload("Files")

	if req.QueryParameter("is_available") != "" {
		q_is_available, err := strconv.ParseBool(req.QueryParameter("is_available"))
		if err == nil {
			tx = tx.Where("is_available = ?", q_is_available)
		}
	}

	if req.QueryParameter("is_accessible") != "" {
		q_is_accessible, err := strconv.ParseBool(req.QueryParameter("is_accessible"))
		if err == nil {
			tx = tx.Where("is_accessible = ?", q_is_accessible)
		}
	}

	if req.QueryParameter("list") == "watchlist" {
		tx = tx.Where("watchlist = ?", true)
	}

	if req.QueryParameter("list") == "favourite" {
		tx = tx.Where("favourite = ?", true)
	}

	q_site := req.QueryParameter("site")
	if q_site != "" {
		tx = tx.Where("site = ?", q_site)
	}

	q_tag := req.QueryParameter("tag")
	if q_tag != "" {
		tx = tx.
			Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
			Joins("left join tags on tags.id=scene_tags.tag_id").
			Where(&Tag{Name: q_tag})
	}

	q_cast := req.QueryParameter("cast")
	if q_cast != "" {
		tx = tx.
			Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
			Joins("left join actors on actors.id=scene_cast.actor_id").
			Where(&Actor{Name: q_cast})
	}

	q_released := req.QueryParameter("released")
	if q_released != "" {
		tx = tx.Where("release_date_text LIKE ?", q_released+"%")
	}

	// Count totals first
	tx.Count(&total)

	// Get scenes
	tx.
		Order("release_date desc").
		Limit(limit).
		Offset(offset).
		Find(&scenes)

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetScenes{Results: total, Scenes: scenes})
}

func (i SceneResource) toggleList(req *restful.Request, resp *restful.Response) {
	var r RequestToggleList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	if r.SceneID == "" && r.List == "" {
		return
	}

	db, _ := GetDB()
	defer db.Close()

	var scene Scene
	err = scene.GetIfExist(r.SceneID)
	if err != nil {
		log.Error(err)
		return
	}

	if r.List == "watchlist" {
		scene.Watchlist = !scene.Watchlist
	}

	if r.List == "favourite" {
		scene.Favourite = !scene.Favourite
	}

	scene.Save()
}
