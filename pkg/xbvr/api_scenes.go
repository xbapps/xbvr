package xbvr

import (
	"net/http"
	"strconv"

	"github.com/blevesearch/bleve"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/markphelps/optional"
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

type RequestSceneList struct {
	Limit        optional.Int      `json:"limit"`
	Offset       optional.Int      `json:"offset"`
	IsAvailable  optional.Bool     `json:"isAvailable"`
	IsAccessible optional.Bool     `json:"isAccessible"`
	IsWatched    optional.Bool     `json:"isWatched"`
	Lists        []optional.String `json:"lists"`
	Sites        []optional.String `json:"sites"`
	Tags         []optional.String `json:"tags"`
	Cast         []optional.String `json:"cast"`
	Cuepoint     []optional.String `json:"cuepoint"`
	Released     optional.String   `json:"releaseMonth"`
	Sort         optional.String   `json:"sort"`
}

type RequestSetSceneRating struct {
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
	tx.Select("strftime('%Y-%m', release_date) as release_date_text").
		Group("strftime('%Y-%m', release_date)").Find(&scenes)
	var outRelease []string
	for i := range scenes {
		outRelease = append(outRelease, scenes[i].ReleaseDateText)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, ResponseGetFilters{Tags: outTags, Cast: outCast, Sites: outSites, ReleaseMonths: outRelease})
}

func (i SceneResource) getScenes(req *restful.Request, resp *restful.Response) {
	var r RequestSceneList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var total = 0

	limit := r.Limit.OrElse(100)
	offset := r.Offset.OrElse(0)

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

	if r.IsAvailable.Present() {
		tx = tx.Where("is_available = ?", r.IsAvailable.OrElse(true))
	}

	if r.IsAccessible.Present() {
		tx = tx.Where("is_accessible = ?", r.IsAccessible.OrElse(true))
	}

	if r.IsWatched.Present() {
		tx = tx.Where("is_watched = ?", r.IsWatched.OrElse(true))
	}

	for _, i := range r.Lists {
		if i.OrElse("") == "watchlist" {
			tx = tx.Where("watchlist = ?", true)
		}
		if i.OrElse("") == "favourite" {
			tx = tx.Where("favourite = ?", true)
		}
	}

	var sites []string
	for _, i := range r.Sites {
		sites = append(sites, i.OrElse(""))
	}
	if len(sites) > 0 {
		tx = tx.Where("site IN (?)", sites)
	}

	var tags []string
	for _, i := range r.Tags {
		tags = append(tags, i.OrElse(""))
	}
	if len(tags) > 0 {
		tx = tx.
			Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
			Joins("left join tags on tags.id=scene_tags.tag_id").
			Where("tags.name IN (?)", tags)
	}

	var cast []string
	for _, i := range r.Cast {
		cast = append(cast, i.OrElse(""))
	}
	if len(cast) > 0 {
		tx = tx.
			Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
			Joins("left join actors on actors.id=scene_cast.actor_id").
			Where("actors.name IN (?)", cast)
	}

	var cuepoint []string
	for _, i := range r.Cuepoint {
		cuepoint = append(cuepoint, i.OrElse(""))
	}
	if len(cuepoint) > 0 {
		tx = tx.Joins("left join scene_cuepoints on scene_cuepoints.scene_id=scenes.id")
		for _, i := range cuepoint {
			tx = tx.Where("scene_cuepoints.name LIKE ?", "%"+i+"%")
		}
	}

	if r.Released.Present() {
		tx = tx.Where("release_date_text LIKE ?", r.Released.OrElse("")+"%")
	}

	switch r.Sort.OrElse("") {
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
	case "scene_added_desc":
		tx = tx.Order("created_at desc")
	case "scene_updated_desc":
		tx = tx.Order("updated_at desc")
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
