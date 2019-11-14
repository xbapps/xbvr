package xbvr

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/models"
)

type RequestMatchFile struct {
	SceneID string `json:"scene_id"`
	FileID  uint   `json:"file_id"`
}

type RequestFileList struct {
	State       optional.String   `json:"state"`
	CreatedDate []optional.String `json:"createdDate"`
	Sort        optional.String   `json:"sort"`
}

type FilesResource struct{}

func (i FilesResource) WebService() *restful.WebService {
	tags := []string{"Files"}

	ws := new(restful.WebService)

	ws.Path("/api/files").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/list").To(i.listFiles).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/match").To(i.matchFile).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/file/{file-id}").To(i.removeFile).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i FilesResource) listFiles(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestFileList
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var files []models.File
	tx := db.Model(&files)

	// State
	switch r.State.OrElse("") {
	case "matched":
		tx = tx.Where("files.scene_id != 0")
	case "unmatched":
		tx = tx.Where("files.scene_id = 0")
	}

	// Creation date
	if len(r.CreatedDate) == 2 {
		t0, _ := time.Parse(time.RFC3339, r.CreatedDate[0].OrElse(""))
		t1, _ := time.Parse(time.RFC3339, r.CreatedDate[1].OrElse(""))
		tx = tx.Where("files.created_time > ? AND files.created_time < ?", t0, t1)
	}

	// Sorting
	switch r.Sort.OrElse("") {
	case "filename_asc":
		tx = tx.Order("filename asc")
	case "filename_desc":
		tx = tx.Order("filename desc")
	case "created_time_asc":
		tx = tx.Order("created_time asc")
	case "created_time_desc":
		tx = tx.Order("created_time desc")
	case "size_asc":
		tx = tx.Order("size asc")
	case "size_desc":
		tx = tx.Order("size desc")
	case "video_height_asc":
		tx = tx.Order("video_height asc")
	case "video_height_desc":
		tx = tx.Order("video_height desc")
	case "video_width_asc":
		tx = tx.Order("video_width asc")
	case "video_width_desc":
		tx = tx.Order("video_width desc")
	case "video_bit_rate_asc":
		tx = tx.Order("video_bit_rate asc")
	case "video_bit_rate_desc":
		tx = tx.Order("video_bit_rate desc")
	}

	tx.Find(&files)

	resp.WriteHeaderAndEntity(http.StatusOK, files)
}

func (i FilesResource) matchFile(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestMatchFile
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

	scene.UpdateStatus()
	db.Close()

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}
