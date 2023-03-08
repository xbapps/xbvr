package api

import (
	"net/http"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/models"
)

type RequestDeleteTagGroup struct {
	TagGroupID uint   `json:"tag_group_id"`
	Name       string `json:"name"`
}

type RequestEditTagGroupMembers struct {
	Name string   `json:"name"`
	Tags []string `json:"tagList"`
}

type ResponseGetTagGroups struct {
	Results int               `json:"results"`
	Scenes  []models.TagGroup `json:"tag_groups"`
}

type ResponseTagGroup struct {
	Status   string          `json:"status"`
	TagGroup models.TagGroup `json:"tag_group"`
}
type ResponseTagGroups struct {
	Error     error             `json:"error"`
	TagGroups []models.TagGroup `json:"tag_groups"`
}

type TagGroupResource struct{}

func (i TagGroupResource) WebService() *restful.WebService {
	tags := []string{"TagGroup"}

	ws := new(restful.WebService)

	ws.Path("/api/tag_group").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/list").To(i.getTagGroups).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ResponseGetScenes{}))

	ws.Route(ws.POST("/create").To(i.createTagGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/delete").To(i.deleteTagGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/{tag-group-name}").To(i.getTagGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/add").To(i.addToTagGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/remove").To(i.removeFromTagGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	ws.Route(ws.POST("/rename").To(i.renameTagGroup).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

	return ws
}

func (i TagGroupResource) createTagGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditTagGroupMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}
	r.Name = strings.TrimSpace(strings.ToLower(r.Name))

	//Construct tag group record
	var tagGroup models.TagGroup
	var tagGroupList []models.Tag
	var tmpTag models.Tag
	var tagGroupTag models.Tag
	var errors []string
	ret := http.StatusOK

	tagCnt := 0
	for _, t := range r.Tags {
		tmpTag.ID = 0
		db.Where("name = ?", t).First(&tmpTag)
		if tmpTag.ID != 0 {
			tagGroupList = append(tagGroupList, tmpTag)
			tagCnt++
		} else {
			errors = append(errors, t+" not found")
			ret = http.StatusNotFound
		}
	}

	if ret != http.StatusNotFound {
		if tagCnt > 0 {
			// create a tag to represent the tag group
			tagGroupTag.ID = 0
			tagGroupTag.Name = "tag group:" + r.Name
			tagGroupTag.Save()

			tagGroup.ID = 0
			tagGroup.TagGroupTag = tagGroupTag
			tagGroup.Tags = tagGroupList
			tagGroup.Name = r.Name
			tagGroup.Save()
		}
	} else {
		createResp := &ResponseTagGroup{
			Status:   strings.Join(errors, ","),
			TagGroup: tagGroup,
		}
		RefreshTagGroup(&tagGroup)
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	RefreshTagGroup(&tagGroup)
	createResp := &ResponseTagGroup{
		Status:   strings.Join(errors, ","),
		TagGroup: tagGroup,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, createResp)
}

func (i TagGroupResource) deleteTagGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestDeleteTagGroup
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var tagGroup models.TagGroup
	if r.TagGroupID != 0 {
		err = db.First(&tagGroup, r.TagGroupID).Error
	} else {
		// find on the tag group name
		err = db.Where(&models.TagGroup{Name: r.Name}).First(&tagGroup).Error
		if err != nil {
			// find on the tag group tag name
			var tag models.Tag
			db.Where("name = ?", r.Name).First(&tag)
			err = db.Where("tag_group_tag_id = ?", tag.ID).First(&tagGroup).Error
		}
	}

	if err != nil {
		log.Error(err)
		resp.WriteHeaderAndEntity(http.StatusNotFound, nil)
		return
	}

	db.Model(&tagGroup).Association("Tags").Clear() // delete tags links with tag group
	db.Delete(&tagGroup)
	defer resp.WriteHeaderAndEntity(http.StatusOK, tagGroup)

	var tagGroupTag models.Tag
	tagGroupTag.ID = tagGroup.TagGroupTagId
	db.Model(&tagGroupTag).Association("Scenes").Clear() // delete scene links with tag group
	db.Delete(&tagGroupTag)                              // delete tag group
	tagGroup.UpdateSceneTagRecords()
}

func (i TagGroupResource) getTagGroups(req *restful.Request, resp *restful.Response) {

	var tagGroups []models.TagGroup
	db, _ := models.GetDB()
	defer db.Close()

	db.Preload("Tags").Find(&tagGroups)
	resp.WriteHeaderAndEntity(http.StatusOK, tagGroups)
}

func (i TagGroupResource) getTagGroup(req *restful.Request, resp *restful.Response) {
	tagGroupName := strings.TrimPrefix(req.PathParameter("tag-group-name"), "tag group:")
	var tagGroup models.TagGroup
	if tagGroupName == "" {
		resp.WriteHeaderAndEntity(http.StatusOK, &ResponseTagGroup{
			Status:   "Specify a Tag Group Name",
			TagGroup: tagGroup,
		})
		return
	}

	db, _ := models.GetDB()
	err := tagGroup.GetIfExistByName(tagGroupName)
	db.Close()

	if err != nil || tagGroup.ID == 0 {
		resp.WriteHeaderAndEntity(http.StatusOK, &ResponseTagGroup{
			Status:   "Tag Group Not Found",
			TagGroup: tagGroup,
		})
	} else {
		resp.WriteHeaderAndEntity(http.StatusOK, &ResponseTagGroup{
			Status:   "",
			TagGroup: tagGroup,
		})
	}
}

func (i TagGroupResource) removeFromTagGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditTagGroupMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var tagGroup models.TagGroup
	var errors []string
	ret := http.StatusOK

	tagGroupName := ""
	for _, t := range r.Tags {
		if strings.HasPrefix(t, "tag group:") {
			tagGroupName = t
		}
	}
	var tag models.Tag
	db.Where("name = ?", tagGroupName).First(&tag)
	db.Where("tag_group_tag_id = ?", tag.ID).Preload("TagGroupTag").Preload("Tags").First(&tagGroup)

	// check we aren't going to remove everyone
	remainTagCount := len(tagGroup.Tags)
	for _, tag := range r.Tags {
		if !strings.HasPrefix(tag, "tag group:") {
			for _, rec := range tagGroup.Tags {
				if rec.Name == tag {
					remainTagCount--
				}
			}
		}
	}
	if remainTagCount < 2 {
		RefreshTagGroup(&tagGroup)
		createResp := &ResponseTagGroup{
			Status:   "A Group needs at least 2 tags. Delete the group instead",
			TagGroup: tagGroup,
		}
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	for _, tag := range r.Tags {
		if !strings.HasPrefix(tag, "tag group:") {
			found := false
			for idx, rec := range tagGroup.Tags {
				if rec.Name == tag {
					db.Model(&tagGroup).Association("Tags").Delete(&tagGroup.Tags[idx])
					found = true
				}
			}
			if !found {
				errors = append(errors, tag+" not found in group")
			}
		}
	}

	tagGroup.Save()

	RefreshTagGroup(&tagGroup)
	if len(errors) > 0 {
		createResp := &ResponseTagGroup{
			Status:   strings.Join(errors, ","),
			TagGroup: tagGroup,
		}
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	createResp := &ResponseTagGroup{
		TagGroup: tagGroup,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, createResp)

}

func (i TagGroupResource) addToTagGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditTagGroupMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var tagGroup models.TagGroup
	var tmpTag models.Tag
	var errors []string
	ret := http.StatusOK

	tagGroupName := ""
	tagCnt := 0
	for _, t := range r.Tags {
		if strings.HasPrefix(t, "tag group:") {
			tagGroupName = t
		}
	}
	var tag models.Tag
	db.Where("name = ?", tagGroupName).First(&tag)
	db.Where("tag_group_tag_id = ?", tag.ID).Preload("TagGroupTag").Preload("TagGroup").First(&tagGroup)

	for _, t := range r.Tags {
		if !strings.HasPrefix(t, "tag group:") {

			tmpTag.ID = 0
			db.Where("name = ?", t).First(&tmpTag)
			if tmpTag.ID != 0 {
				// check if the tag is already this group
				cnt := 0
				db.Model(&tagGroup).
					Joins("join tag_group_tags on tag_group_tags.tag_group_id = tag_groups.id").
					Where("tag_group_tags.tag_id  = ?", tmpTag.ID).Count(&cnt)
				if cnt > 0 {
					errors = append(errors, t+" already in this group")
				} else {
					tagGroup.Tags = append(tagGroup.Tags, tmpTag)
					tagCnt++
				}
			} else {
				errors = append(errors, t+" not found")
			}
		}
	}

	if tagCnt > 0 {
		tagGroup.Save()
	}

	RefreshTagGroup(&tagGroup)
	if len(errors) > 0 {
		createResp := &ResponseTagGroup{
			Status:   strings.Join(errors, ","),
			TagGroup: tagGroup,
		}
		resp.WriteHeaderAndEntity(ret, createResp)
		return
	}

	createResp := &ResponseTagGroup{
		TagGroup: tagGroup,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, createResp)

}
func (i TagGroupResource) renameTagGroup(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	//Get request data
	var r RequestEditTagGroupMembers
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	r.Name = strings.TrimSpace(strings.ToLower(r.Name))

	var tagGroup models.TagGroup
	cnt := 0
	db.Where("name = ?", r.Name).First(&tagGroup).Count(&cnt)
	if cnt > 0 {
		createResp := &ResponseTagGroup{
			Status:   "Error: Tag Group " + r.Name + " already exists",
			TagGroup: tagGroup,
		}
		resp.WriteHeaderAndEntity(http.StatusOK, createResp)
		return
	}

	db.Where("name = ?", strings.TrimPrefix(r.Tags[0], "tag group:")).Preload("TagGroupTag").First(&tagGroup)
	tagGroup.TagGroupTag.Name = "tag group:" + r.Name
	tagGroup.Name = r.Name
	tagGroup.Save()

	createResp := &ResponseTagGroup{
		Status:   "",
		TagGroup: tagGroup,
	}
	resp.WriteHeaderAndEntity(http.StatusOK, createResp)
}

func RefreshTagGroup(tagGroup *models.TagGroup) {
	db, _ := models.GetDB()
	defer db.Close()

	tagGroup.UpdateSceneTagRecords()
	db.Preload("TagGroupTag").Preload("Tags").Find(&tagGroup)
}
