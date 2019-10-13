package xbvr

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/xbapps/xbvr/pkg/models"
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

	ws.Route(ws.DELETE("/file/{file-id}").To(i.removeFile).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i FilesResource) listUnmatchedFiles(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var files []models.File
	db.Raw(`select files.* from files where files.scene_id = 0;`).Scan(&files)

	resp.WriteHeaderAndEntity(http.StatusOK, files)
}

func (i FilesResource) matchFile(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r MatchFileRequest
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	// Assign Scene to File
	var scene models.Scene
	err = scene.GetIfExist(r.SceneID)
	if err != nil {
		log.Error(err)
		return
	}

	var f models.File
	err = db.Where(&models.File{ID: r.FileID}).First(&f).Error
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
	}

	// Finally, update scene available/accessible status
	scene.IsAvailable = true
	if f.Exists() {
		scene.IsAccessible = true
	}
	scene.Save()

	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}

func (i FilesResource) removeFile(req *restful.Request, resp *restful.Response) {
	fileId, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		return
	}

	var file models.File
	var scene models.Scene
	db, _ := models.GetDB()
	err = db.Where(&models.File{ID: uint(fileId)}).First(&file).Error
	if err == nil {
		err := os.Remove(filepath.Join(file.Path, file.Filename))
		if err == nil {
			db.Delete(&file)
		} else {
			log.Errorf("Error deleting file ", err)
		}
		if file.SceneID != 0 {
			scene.GetIfExistByPK(file.SceneID)
		}
	} else {
		log.Errorf("Error deleting file ", err)
	}
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}
