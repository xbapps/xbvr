package xbase

import (
	"net/http"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
)

type ExtScene struct {
	SceneID     string   `json:"_id"`
	SiteID      string   `json:"scene_id"`
	SceneType   string   `json:"scene_type"`
	Title       string   `json:"title"`
	Studio      string   `json:"studio"`
	Site        string   `json:"site"`
	Covers      []string `json:"covers"`
	Gallery     []string `json:"gallery"`
	Tags        []string `json:"tags"`
	Cast        []string `json:"cast"`
	Filenames   []string `json:"filename"`
	Duration    int      `json:"duration"`
	Synopsis    string   `json:"synopsis"`
	Released    string   `json:"released"`
	HomepageURL string   `json:"homepage_url"`
}

type ExtSceneResponse struct {
	Status string      `json:"status"`
	Scene  interface{} `json:"scene"`
}

type ExtResource struct{}

func (i ExtResource) WebService() *restful.WebService {
	tags := []string{"Ext"}

	ws := new(restful.WebService)

	ws.Path("/api/ext").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/status").To(i.status).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/scene/{scene-id}").To(i.checkScene).
		Param(ws.PathParameter("scene-id", "Scene ID").DataType("string")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(ExtSceneResponse{}))

	ws.Route(ws.POST("/scene").To(i.createScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(ExtScene{}).
		Writes(ExtSceneResponse{}).
		Returns(http.StatusCreated, "Created", Scene{}).
		Returns(http.StatusConflict, "Already exist", ExtSceneResponse{}))

	return ws
}

func (i ExtResource) status(req *restful.Request, resp *restful.Response) {

}

func (i ExtResource) checkScene(req *restful.Request, resp *restful.Response) {
	id := req.PathParameter("scene-id")

	// Check if scene exist
	db, _ := GetDB()
	defer db.Close()

	localScene := Scene{}
	err := localScene.GetIfExist(id)

	// Output
	out := ExtScene{}
	SceneToExt(localScene, &out)

	if err == gorm.ErrRecordNotFound {
		resp.WriteHeaderAndEntity(http.StatusNotFound, ExtSceneResponse{Status: "not-found"})
		return
	}

	resp.WriteHeaderAndEntity(http.StatusConflict, ExtSceneResponse{Status: "exist", Scene: out})
}

func (i ExtResource) createScene(req *restful.Request, resp *restful.Response) {
	obj := ExtScene{}
	err := req.ReadEntity(&obj)
	if err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	// Check if scene exist
	localScene := Scene{}
	err = localScene.GetIfExist(obj.SceneID)

	// Output
	// out := ExtScene{}
	// SceneToExt(localScene, &out)
	// if err == nil {
	// 	resp.WriteHeaderAndEntity(http.StatusConflict, ExtSceneResponse{Status: "exist", Scene: out})
	// 	return
	// }

	// Save
	db, _ := GetDB()
	defer db.Close()

	localScene = Scene{}
	db.Where(&Scene{SceneID: obj.SceneID}).FirstOrCreate(&localScene)

	localScene.SceneID = obj.SceneID
	localScene.Title = obj.Title
	localScene.SceneType = obj.SceneType
	localScene.Studio = obj.Studio
	localScene.Site = obj.Site
	localScene.Duration = obj.Duration
	localScene.Synopsis = obj.Synopsis
	localScene.ReleaseDateText = obj.Released
	localScene.CoverURL = obj.Covers[0]
	localScene.SceneURL = obj.HomepageURL

	if obj.Released != "" {
		dateParsed, err := dateparse.ParseLocal(strings.Replace(obj.Released, ",", "", -1))
		if err == nil {
			localScene.ReleaseDate = dateParsed
		}
	}

	db.Save(localScene)

	// Associate Tags
	var tmpTag Tag
	for _, name := range obj.Tags {
		tagClean := convertTag(name)
		if tagClean != "" {
			tmpTag = Tag{}
			db.Where(&Tag{Name: tagClean}).FirstOrCreate(&tmpTag)
			db.Model(&localScene).Association("Tags").Append(tmpTag)
		}
	}

	// Associate Actors
	var tmpActor Actor
	for _, name := range obj.Cast {
		tmpActor = Actor{}
		db.Where(&Actor{Name: name}).FirstOrCreate(&tmpActor)
		db.Model(&localScene).Association("Cast").Append(tmpActor)
	}

	// Associate Filenames
	var tmpSceneFilename PossibleFilename
	for _, name := range obj.Filenames {
		tmpSceneFilename = PossibleFilename{}
		db.Where(&PossibleFilename{Name: name}).FirstOrCreate(&tmpSceneFilename)
		db.Model(&localScene).Association("Filenames").Append(tmpSceneFilename)
	}

	// Associate Images (but first remove old ones)
	db.Unscoped().Where(&Image{SceneID: localScene.ID}).Delete(Image{})

	for _, u := range obj.Covers {
		tmpImage := Image{}
		db.Where(&Image{URL: u}).FirstOrCreate(&tmpImage)
		tmpImage.SceneID = localScene.ID
		tmpImage.Type = "cover"
		tmpImage.Save()
	}

	for _, u := range obj.Gallery {
		tmpImage := Image{}
		db.Where(&Image{URL: u}).FirstOrCreate(&tmpImage)
		tmpImage.SceneID = localScene.ID
		tmpImage.Type = "gallery"
		tmpImage.Save()
	}

	resp.WriteHeader(http.StatusOK)
}

func SceneToExt(in Scene, out *ExtScene) {
	out.SceneID = in.SceneID
	out.SiteID = ""
	out.SceneType = in.SceneType
	out.Title = in.Title
	out.Studio = in.Studio
	out.Site = in.Site
	out.Duration = in.Duration
	out.Synopsis = in.Synopsis
	out.Released = in.ReleaseDateText // TODO: convert
	out.HomepageURL = in.SceneURL
	// out.Covers
	// out.Tags
	// out.Cast
	// out.Filenames
}
