package xbvr

import (
	"encoding/json"
	"net/http"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type MatchFileRequest struct {
	SceneID string `json:"scene_id"`
	FileID  uint   `json:"file_id"`
}

type FilesResource struct{}

func (i FilesResource) WebService() *restful.WebService {
	tags := []string{"Files"}

	ws := new(restful.WebService)

	ws.Path("/api/files").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/list/unmatched").To(i.listUnmatchedFiles).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/match").To(i.matchFile).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i FilesResource) listUnmatchedFiles(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Raw(`select files.* from files where files.scene_id = 0;`).Scan(&files)

	resp.WriteHeaderAndEntity(http.StatusOK, files)
}

func (i FilesResource) matchFile(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	var r MatchFileRequest
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	// Assign Scene to File
	var scene Scene
	err = scene.GetIfExist(r.SceneID)
	if err != nil {
		log.Error(err)
		return
	}

	var f File
	err = db.Where(&File{ID: r.FileID}).First(&f).Error
	if err == nil {
		f.SceneID = scene.ID
		f.Save()
	}

	// Add File to the list of Scene filenames so it will be discovered when file is moved
	var pfTxt []string
	err = json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt)
	if err != nil {
		log.Error(err)
		return
	}

	pfTxt = append(pfTxt, f.Filename)
	tmp, err := json.Marshal(pfTxt)
	if err == nil {
		scene.FilenamesArr = string(tmp)
		scene.Save()
	}

	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}
