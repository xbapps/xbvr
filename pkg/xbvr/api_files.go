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

type UnMatchFileRequest struct {
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

	ws.Route(ws.POST("/unmatch").To(i.unMatchFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Scene{}))

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

func (i FilesResource) unMatchFile(req *restful.Request, resp *restful.Response) {
	var r UnMatchFileRequest
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var f models.File
	err = f.GetIfExist(r.FileID)
	if err != nil {
		log.Error(err)
		return
	}

	sceneID := f.SceneID

	f.SceneID = 0
	f.Save()

	var scene models.Scene
	err = scene.GetIfExistByPK(sceneID)
	if err != nil {
		log.Error(err)
		return
	}

	var pfTxt []string
	err = json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt)
	if err != nil {
		log.Error(err)
		return
	}

	var newFilenamesArr []string
	for _, fn := range pfTxt {
		if fn != f.Filename {
			newFilenamesArr = append(newFilenamesArr, fn)
		}
	}

	tmp, err := json.Marshal(newFilenamesArr)
	if err != nil {
		log.Error(err)
		return
	}

	scene.FilenamesArr = string(tmp)
	scene.Save()

	files, err := scene.GetFiles()
	if err != nil {
		return
	}

	if len(files) == 0 {
		scene.IsAvailable = false
		scene.IsAccessible = false
		scene.Save()
	}

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
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
	err = f.GetIfExist(r.FileID)
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
	err = file.GetIfExist(uint(fileId))
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
