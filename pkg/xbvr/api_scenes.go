package xbvr

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/thoas/go-funk"
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

type RequestSceneRating struct {
	Rating float64 `json:"rating"`
}

type ResponseGetScenes struct {
	Results int            `json:"results"`
	Scenes  []models.Scene `json:"scenes"`
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

	ws.Route(ws.POST("/cuepoint/{scene-id}").To(i.addSceneCuepoint).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/rate/{scene-id}").To(i.rateScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/toggle").To(i.toggleList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.GET("/search").To(i.searchSceneIndex).
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
	db, _ := models.GetDB()
	defer db.Close()

	var tags []models.Tag
	db.Model(&models.Tag{}).Order("name").Find(&tags)

	var outTags []string
	for i := range tags {
		outTags = append(outTags, tags[i].Name)
	}

	var actors []models.Actor
	db.Model(&models.Actor{}).Order("name").Find(&actors)

	var outCast []string
	for i := range actors {
		outCast = append(outCast, actors[i].Name)
	}

	var scenes []models.Scene
	db.Model(&models.Scene{}).Order("site").Group("site").Find(&scenes)

	var outSites []string
	for i := range scenes {
		if scenes[i].Site != "" {
			outSites = append(outSites, scenes[i].Site)
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetFilters{Tags: outTags, Cast: outCast, Sites: outSites})
}

func (i SceneResource) getFiltersForState(req *restful.Request, resp *restful.Response) {
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

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	tx := db.
		Model(&scenes).
		Preload("Cast").
		Preload("Tags").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints")

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

	q_is_watched := req.QueryParameter("is_watched")
	switch q_is_watched {
	case "1":
		tx = tx.Where("is_watched = ?", true)
	case "0":
		tx = tx.Where("is_watched = ?", false)
	default:
	}

	q_lists := req.QueryParameter("lists")
	if q_lists != "" {
		if funk.ContainsString(strings.Split(q_lists, ","), "watchlist") {
			tx = tx.Where("watchlist = ?", true)
		}
		if funk.ContainsString(strings.Split(q_lists, ","), "favourite") {
			tx = tx.Where("favourite = ?", true)
		}
	}

	q_sites := req.QueryParameter("sites")
	if q_sites != "" {
		tx = tx.Where("site IN (?)", strings.Split(q_sites, ","))
	}

	q_tags := req.QueryParameter("tags")
	if q_tags != "" {
		tx = tx.
			Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
			Joins("left join tags on tags.id=scene_tags.tag_id").
			Where("tags.name IN (?)", strings.Split(q_tags, ","))
	}

	q_cast := req.QueryParameter("cast")
	if q_cast != "" {
		tx = tx.
			Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
			Joins("left join actors on actors.id=scene_cast.actor_id").
			Where("actors.name IN (?)", strings.Split(q_cast, ","))
	}

	q_released := req.QueryParameter("released")
	if q_released != "" {
		tx = tx.Where("release_date_text LIKE ?", q_released+"%")
	}

	q_sort := req.QueryParameter("sort")
	switch q_sort {
	case "added_desc":
		tx = tx.Order("added_date desc")
	case "added_asc":
		tx = tx.Order("added_date asc")
	case "release_desc":
		tx = tx.Order("release_date desc")
	case "release_asc":
		tx = tx.Order("release_date asc")
	case "rating_desc":
		tx = tx.
			Where("star_rating > ?", 0).
			Order("star_rating desc")
	case "rating_asc":
		tx = tx.
			Where("star_rating > ?", 0).
			Order("star_rating asc")
	case "last_opened":
		tx = tx.
			Where("last_opened > ?", "0001-01-01 00:00:00+00:00").
			Order("last_opened desc")
	case "random":
		tx = tx.Order("random()")
	default:
		tx = tx.Order("release_date desc")
	}

	// Count totals first
	tx.
		Group("scenes.scene_id").
		Count(&total)

	// Get scenes
	tx.
		Group("scenes.scene_id").
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

func (i SceneResource) searchSceneDB(req *restful.Request, resp *restful.Response) {
	q := "%" + req.QueryParameter("q") + "%"

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Raw(`select scenes.*
			from scenes
					 left join scene_tags on scene_tags.scene_id = scenes.id
					 left join tags on tags.id = scene_tags.tag_id
					 left join scene_cast on scene_cast.scene_id = scenes.id
					 left join actors on actors.id = scene_cast.actor_id
			where tags.name like ?
			   or actors.name like ?
			   or scenes.title like ?
			   or scenes.synopsis like ?
			   or scenes.scene_id like ?
			   or scenes.filenames_arr like ?
               or scenes.site like ?
			group by scenes.scene_id;`, q, q, q, q, q, q, q).Scan(&scenes)

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetScenes{Results: len(scenes), Scenes: scenes})
}

func (i SceneResource) searchSceneIndex(req *restful.Request, resp *restful.Response) {
	q := req.QueryParameter("q")
	q = strings.Replace(q, ".", " ", -1)
	q = strings.Replace(q, "_", " ", -1)
	q = strings.Replace(q, "+", " ", -1)
	q = strings.Replace(q, "-", " ", -1)

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

	var r RequestSceneRating
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
