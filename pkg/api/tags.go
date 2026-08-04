package api

import (
	"net/http"
	"strconv"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/models"
)

type ResponseGetTags struct {
	Results int          `json:"results"`
	Tags    []models.Tag `json:"tags"`
}

type RequestPromoteTag struct {
	IsPromoted bool `json:"is_promoted"`
}

type TagResource struct{}

func (i TagResource) WebService() *restful.WebService {
	tags := []string{"Tag"}

	ws := new(restful.WebService)

	ws.Path("/api/tag").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/list").To(i.getTagList).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetTags{}))

	ws.Route(ws.POST("/promote/{tag-id}").To(i.promoteTag).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Tag{}))

	return ws
}

func (i TagResource) getTagList(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var tags []models.Tag
	db.Model(&models.Tag{}).
		Select("id, name, is_promoted").
		Order("name").
		Find(&tags)

	type sceneCount struct {
		TagID uint
		Count int
	}
	var counts []sceneCount
	db.Table("scenes").
		Select("scene_tags.tag_id, count(*) as count").
		Joins("join scene_tags on scene_tags.scene_id = scenes.id").
		Where("scenes.deleted_at is null").
		Group("scene_tags.tag_id").
		Scan(&counts)

	countMap := make(map[uint]int, len(counts))
	for _, c := range counts {
		countMap[c.TagID] = c.Count
	}
	for i := range tags {
		tags[i].Count = countMap[tags[i].ID]
	}

	out := ResponseGetTags{
		Results: len(tags),
		Tags:    tags,
	}
	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func (i TagResource) promoteTag(req *restful.Request, resp *restful.Response) {
	tagID, err := strconv.Atoi(req.PathParameter("tag-id"))
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, err)
		return
	}

	var r RequestPromoteTag
	err = req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var tag models.Tag
	if err := db.First(&tag, tagID).Error; err != nil {
		resp.WriteHeaderAndEntity(http.StatusNotFound, err)
		return
	}

	tag.IsPromoted = r.IsPromoted
	if err := db.Save(&tag).Error; err != nil {
		log.Error(err)
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, err)
		return
	}

	resp.WriteHeaderAndEntity(http.StatusOK, tag)
}
