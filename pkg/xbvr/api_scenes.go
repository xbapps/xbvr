package xbvr

import (
	"net/http"
	"strconv"

	"github.com/blevesearch/bleve"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/xbapps/xbvr/pkg/models"
)

type RequestToggleList struct {
	SceneID string `json:"scene_id"`
	List    string `json:"list"`
}

type RequestSceneCuepoint struct {
	TimeStart float64 `json:"time_start"`
	Name      string  `json:"name"`
}

type RequestSetSceneRating struct {
	Rating float64 `json:"rating"`
}

type ResponseGetScenes struct {
	Results int            `json:"results"`
	Scenes  []models.Scene `json:"scenes"`
}

type ResponseGetFilters struct {
	Cast          []string        `json:"cast"`
	Tags          []string        `json:"tags"`
	Sites         []string        `json:"sites"`
	ReleaseMonths []string        `json:"release_month"`
	Volumes       []models.Volume `json:"volumes"`
}

type SceneResource struct{}

func (i SceneResource) WebService() *restful.WebService {
	tags := []string{"Scene"}

	ws := new(restful.WebService)

	ws.Path("/api/scene").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/filters").To(i.getFilters).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetFilters{}))

	ws.Route(ws.POST("/list").To(i.getScenes).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.GET("/search").To(i.searchSceneIndex).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.POST("/cuepoint/{scene-id}").To(i.addSceneCuepoint).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/rate/{scene-id}").To(i.rateScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/toggle").To(i.toggleList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	return ws
}

func (i SceneResource) getFilters(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	// Get all accessible scenes
	var scenes []models.Scene
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
	switch db.Dialect().GetName() {
	case "mysql":
		tx.Select("DATE_FORMAT(release_date, '%Y-%m') as release_date_text").
			Group("DATE_FORMAT(release_date, '%Y-%m')").Find(&scenes)
	case "sqlite3":
		tx.Select("strftime('%Y-%m', release_date) as release_date_text").
			Group("strftime('%Y-%m', release_date)").Find(&scenes)
	}
	var outRelease []string
	for i := range scenes {
		outRelease = append(outRelease, scenes[i].ReleaseDateText)
	}

	// Volumes
	var outVolumes []models.Volume
	db.Model(&models.Volume{}).Find(&outVolumes)

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetFilters{
		Tags:          outTags,
		Cast:          outCast,
		Sites:         outSites,
		ReleaseMonths: outRelease,
		Volumes:       outVolumes,
	})
}

func (i SceneResource) getScenes(req *restful.Request, resp *restful.Response) {
	var r models.RequestSceneList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	out := models.QueryScenes(r, true)
	resp.WriteHeaderAndEntity(http.StatusOK, out)
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

	db, _ := models.GetDB()
	defer db.Close()

	var scene models.Scene
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

func (i SceneResource) searchSceneIndex(req *restful.Request, resp *restful.Response) {
	q := req.QueryParameter("q")

	db, _ := models.GetDB()
	defer db.Close()

	idx := NewIndex("scenes")
	defer idx.bleve.Close()
	query := bleve.NewQueryStringQuery(q)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"fulltext"}
	searchRequest.IncludeLocations = true
	searchRequest.From = 0
	searchRequest.Size = 25
	searchRequest.SortBy([]string{"-_score"})

	searchResults, err := idx.bleve.Search(searchRequest)
	if err != nil {
		log.Error(err)
		return
	}

	var scenes []models.Scene
	for _, v := range searchResults.Hits {
		var scene models.Scene
		err := scene.GetIfExist(v.ID)
		if err != nil {
			continue
		}

		scene.Score = v.Score
		scenes = append(scenes, scene)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetScenes{Results: len(scenes), Scenes: scenes})
}

func (i SceneResource) addSceneCuepoint(req *restful.Request, resp *restful.Response) {
	sceneId, err := strconv.Atoi(req.PathParameter("scene-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var r RequestSceneCuepoint
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var scene models.Scene
	db, _ := models.GetDB()
	err = scene.GetIfExistByPK(uint(sceneId))
	if err == nil {
		t := models.SceneCuepoint{
			SceneID:   scene.ID,
			TimeStart: r.TimeStart,
			Name:      r.Name,
		}
		t.Save()

		scene.GetIfExistByPK(uint(sceneId))
	}
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i SceneResource) rateScene(req *restful.Request, resp *restful.Response) {
	sceneId, err := strconv.Atoi(req.PathParameter("scene-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var r RequestSetSceneRating
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var scene models.Scene
	db, _ := models.GetDB()
	err = scene.GetIfExistByPK(uint(sceneId))
	if err == nil {
		scene.StarRating = r.Rating
		scene.Save()
	}
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}
