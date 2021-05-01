package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-test/deep"
	"github.com/xbapps/xbvr/pkg/tasks"

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

type RequestSelectScript struct {
	FileID uint `json:"file_id"`
}

type RequestEditSceneDetails struct {
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	Studio       string   `json:"studio"`
	Site         string   `json:"site"`
	SceneURL     string   `json:"scene_url"`
	ReleaseDate  string   `json:"release_date_text"`
	Cast         []string `json:"castArray"`
	Tags         []string `json:"tagsArray"`
	FilenamesArr string   `json:"filenames_arr"`
	Images       string   `json:"images"`
	CoverURL     string   `json:"cover_url"`
	IsMultipart  bool     `json:"is_multipart"`
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

	ws.Route(ws.POST("/selectscript/{scene-id}").To(i.selectScript).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/edit/{scene-id}").To(i.editScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/toggle").To(i.toggleList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.GET("/{scene-id}").To(i.getScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

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

func (i SceneResource) getScene(req *restful.Request, resp *restful.Response) {
	sceneId, err := strconv.Atoi(req.PathParameter("scene-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var scene models.Scene
	db, _ := models.GetDB()
	err = scene.GetIfExistByPK(uint(sceneId))
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
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

	idx := tasks.NewIndex("scenes")
	defer idx.Bleve.Close()
	query := bleve.NewQueryStringQuery(q)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"fulltext"}
	searchRequest.IncludeLocations = true
	searchRequest.From = 0
	searchRequest.Size = 25
	searchRequest.SortBy([]string{"-_score"})

	searchResults, err := idx.Bleve.Search(searchRequest)
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

func (i SceneResource) selectScript(req *restful.Request, resp *restful.Response) {
	sceneId, err := strconv.Atoi(req.PathParameter("scene-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var r RequestSelectScript
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var scene models.Scene
	var files []models.File
	db, _ := models.GetDB()
	err = scene.GetIfExistByPK(uint(sceneId))
	if err == nil {
		files, err = scene.GetScriptFiles()
		if err == nil {
			for _, file := range files {
				if file.ID == r.FileID && !file.IsSelectedScript {
					file.IsSelectedScript = true
					file.Save()
				} else if file.ID != r.FileID && file.IsSelectedScript {
					file.IsSelectedScript = false
					file.Save()
				}
			}
		}
		err = scene.GetIfExistByPK(uint(sceneId))
	}
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i SceneResource) editScene(req *restful.Request, resp *restful.Response) {
	sceneId, err := strconv.Atoi(req.PathParameter("scene-id"))
	if err != nil {
		log.Error(err)
		return
	}

	var r RequestEditSceneDetails
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var scene models.Scene
	db, _ := models.GetDB()
	defer db.Close()
	err = scene.GetIfExistByPK(uint(sceneId))
	if err == nil {
		if scene.Title != r.Title {
			scene.Title = r.Title
			models.AddAction(scene.SceneID, "edit", "title", r.Title)
		}
		if scene.Synopsis != r.Synopsis {
			scene.Synopsis = r.Synopsis
			models.AddAction(scene.SceneID, "edit", "synopsis", r.Synopsis)
		}
		if scene.Studio != r.Studio {
			scene.Studio = r.Studio
			models.AddAction(scene.SceneID, "edit", "studio", r.Studio)
		}
		if scene.Site != r.Site {
			scene.Site = r.Site
			models.AddAction(scene.SceneID, "edit", "site", r.Site)
		}
		if scene.SceneURL != r.SceneURL {
			scene.SceneURL = r.SceneURL
			models.AddAction(scene.SceneID, "edit", "scene_url", r.SceneURL)
		}
		if scene.ReleaseDateText != r.ReleaseDate {
			scene.ReleaseDateText = r.ReleaseDate
			scene.ReleaseDate, _ = time.Parse("2006-01-02", r.ReleaseDate)
			models.AddAction(scene.SceneID, "edit", "release_date_text", r.ReleaseDate)
		}
		if scene.FilenamesArr != r.FilenamesArr {
			scene.FilenamesArr = r.FilenamesArr
			models.AddAction(scene.SceneID, "edit", "filenames_arr", r.FilenamesArr)
		}
		if scene.Images != r.Images {
			scene.Images = r.Images
			models.AddAction(scene.SceneID, "edit", "images", r.Images)
		}
		if scene.CoverURL != r.CoverURL {
			scene.CoverURL = r.CoverURL
			models.AddAction(scene.SceneID, "edit", "cover_url", r.CoverURL)
		}
		if scene.IsMultipart != r.IsMultipart {
			scene.IsMultipart = r.IsMultipart
			models.AddAction(scene.SceneID, "edit", "is_multipart", strconv.FormatBool(r.IsMultipart))
		}

		var diffs []string

		newTags := make([]models.Tag, 0)
		for _, v := range r.Tags {
			nt := models.Tag{}
			tagClean := models.ConvertTag(v)
			if tagClean != "" {
				db.Where(&models.Tag{Name: tagClean}).FirstOrCreate(&nt)
				newTags = append(newTags, nt)
			}
		}

		diffs = deep.Equal(scene.Tags, newTags)
		if len(diffs) > 0 {
			exactDifferences := getTagDifferences(scene.Tags, newTags)
			for _, v := range exactDifferences {
				models.AddAction(scene.SceneID, "edit", "tags", v)
			}

			for _, v := range scene.Tags {
				db.Model(&scene).Association("Tags").Delete(&v)
			}

			for _, v := range newTags {
				db.Model(&scene).Association("Tags").Append(&v)
			}
		}

		newCast := make([]models.Actor, 0)
		for _, v := range r.Cast {
			nc := models.Actor{}
			db.Where(&models.Actor{Name: v}).FirstOrCreate(&nc)
			newCast = append(newCast, nc)
		}

		diffs = deep.Equal(scene.Cast, newCast)
		if len(diffs) > 0 {
			exactDifferences := getCastDifferences(scene.Cast, newCast)
			for _, v := range exactDifferences {
				models.AddAction(scene.SceneID, "edit", "cast", v)
			}

			for _, v := range scene.Cast {
				db.Model(&scene).Association("Cast").Delete(&v)
			}

			for _, v := range newCast {
				db.Model(&scene).Association("Cast").Append(&v)
			}
		}

		scene.Save()

		resp.WriteHeaderAndEntity(http.StatusOK, scene)
	}
}

func getTagDifferences(arr1, arr2 []models.Tag) []string {
	output := make([]string, 0)
	for _, v := range arr1 {
		if !tagsContains(arr2, v) {
			output = append(output, fmt.Sprintf("-%v", v.Name))
		}
	}
	for _, v := range arr2 {
		if !tagsContains(arr1, v) {
			output = append(output, fmt.Sprintf("+%v", v.Name))
		}
	}
	return output
}

func getCastDifferences(arr1, arr2 []models.Actor) []string {
	output := make([]string, 0)
	for _, v := range arr1 {
		if !castContains(arr2, v) {
			output = append(output, fmt.Sprintf("-%v", v.Name))
		}
	}
	for _, v := range arr2 {
		if !castContains(arr1, v) {
			output = append(output, fmt.Sprintf("+%v", v.Name))
		}
	}
	return output
}

func tagsContains(arr []models.Tag, val interface{}) bool {
	for _, v := range arr {
		diffs := deep.Equal(v, val)
		if len(diffs) == 0 {
			return true
		}
	}
	return false
}

func castContains(arr []models.Actor, val interface{}) bool {
	for _, v := range arr {
		diffs := deep.Equal(v, val)
		if len(diffs) == 0 {
			return true
		}
	}
	return false
}
