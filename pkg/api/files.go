package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

type RequestUnmatchFile struct {
	FileID uint `json:"file_id"`
}

type RequestFileList struct {
	State       optional.String   `json:"state"`
	CreatedDate []optional.String `json:"createdDate"`
	Sort        optional.String   `json:"sort"`
	Resolutions []optional.String `json:"resolutions"`
	Framerates  []optional.String `json:"framerates"`
	Bitrates    []optional.String `json:"bitrates"`
	Filename    optional.String   `json:"filename"`
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

	ws.Route(ws.POST("/unmatch").To(i.unmatchFile).
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

	// Resolution
	resolutionClauses := []string{}
	if len(r.Resolutions) > 0 {
		for _, resolution := range r.Resolutions {
			if resolution.OrElse("") == "below4k" {
				resolutionClauses = append(resolutionClauses, "video_height between 0 and 1899")
			}
			if resolution.OrElse("") == "4k" {
				resolutionClauses = append(resolutionClauses, "video_height between 1900 and 2449")
			}
			if resolution.OrElse("") == "5k" {
				resolutionClauses = append(resolutionClauses, "video_height between 2450 and 2899")
			}
			if resolution.OrElse("") == "6k" {
				resolutionClauses = append(resolutionClauses, "video_height between 2900 and 3299")
			}
			if resolution.OrElse("") == "above6k" {
				resolutionClauses = append(resolutionClauses, "video_height between 3300 and 9999")
			}
		}
		tx = tx.Where("(" + strings.Join(resolutionClauses, " OR ") + ") AND video_height != 0")
	}

	// Bitrate
	bitrateClauses := []string{}
	if len(r.Bitrates) > 0 {
		for _, bitrate := range r.Bitrates {
			if bitrate.OrElse("") == "low" {
				bitrateClauses = append(bitrateClauses, "video_bit_rate between 0 and 14999999")
			}
			if bitrate.OrElse("") == "medium" {
				bitrateClauses = append(bitrateClauses, "video_bit_rate between 15000000 and 24999999")
			}
			if bitrate.OrElse("") == "high" {
				bitrateClauses = append(bitrateClauses, "video_bit_rate between 25000000 and 35000000")
			}
			if bitrate.OrElse("") == "ultra" {
				bitrateClauses = append(bitrateClauses, "video_bit_rate between 35000001 and 999999999")
			}
		}
		tx = tx.Where("(" + strings.Join(bitrateClauses, " OR ") + ") AND video_bit_rate != 0")
	}

	// Framerate
	framerateClauses := []string{}
	if len(r.Framerates) > 0 {
		for _, framerate := range r.Framerates {
			if framerate.OrElse("") == "30fps" {
				framerateClauses = append(framerateClauses, "video_avg_frame_rate_val = 30.0")
			}
			if framerate.OrElse("") == "60fps" {
				framerateClauses = append(framerateClauses, "video_avg_frame_rate_val = 60.0")
			}
			if framerate.OrElse("") == "other" {
				framerateClauses = append(framerateClauses, "(video_avg_frame_rate_val != 30.0 AND video_avg_frame_rate_val != 60.0)")
			}
		}
		tx = tx.Where("(" + strings.Join(framerateClauses, " OR ") + ") AND video_avg_frame_rate_val != 0")
	}

	// Filename
	if len(r.Filename.OrElse("")) > 0 {
		tx = tx.Where("filename like ?", "%"+r.Filename.OrElse("")+"%")
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
	case "duration_asc":
		tx = tx.Order("video_duration asc")
	case "duration_desc":
		tx = tx.Order("video_duration desc")
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
	case "video_bitrate_asc":
		tx = tx.Order("video_bit_rate asc")
	case "video_bitrate_desc":
		tx = tx.Order("video_bit_rate desc")
	case "video_avgfps_val_asc":
		tx = tx.Order("video_avg_frame_rate_val asc")
	case "video_avgfps_val_desc":
		tx = tx.Order("video_avg_frame_rate_val desc")
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
	err = db.Preload("Volume").Where(&models.File{ID: r.FileID}).First(&f).Error
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

	models.AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)

	// Finally, update scene available/accessible status
	scene.UpdateStatus()

	resp.WriteHeaderAndEntity(http.StatusOK, nil)
}

func (i FilesResource) unmatchFile(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var r RequestUnmatchFile
	err := req.ReadEntity(&r)
	if err != nil {
		log.Error(err)
		return
	}

	var f models.File
	err = db.Preload("Volume").Where(&models.File{ID: r.FileID}).First(&f).Error
	var sceneID uint = 0
	if err == nil {
		sceneID = f.SceneID
		if sceneID != 0 {
			f.SceneID = 0
			f.Save()
		}

	}

	var scene models.Scene
	if sceneID != 0 {
		err = scene.GetIfExistByPK(sceneID)
		if err != nil {
			log.Error(err)
			return
		}

		// Remove File from the list of Scene filenames so it will be not be auto-matched again
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
		if err == nil {
			scene.FilenamesArr = string(tmp)
		}

		models.AddAction(scene.SceneID, "unmatch", "filenames_arr", scene.FilenamesArr)

		// Finally, update scene available/accessible status
		scene.UpdateStatus()
	}

	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i FilesResource) removeFile(req *restful.Request, resp *restful.Response) {
	fileId, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		return
	}
	scene := removeFileByFileId(uint(fileId))
	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}
func removeFileByFileId(fileId uint) models.Scene {

	var scene models.Scene
	var file models.File
	db, _ := models.GetDB()
	defer db.Close()

	err := db.Preload("Volume").Where(&models.File{ID: fileId}).First(&file).Error
	if err == nil {

		deleted := false
		switch file.Volume.Type {
		case "local":
			err := os.Remove(filepath.Join(file.Path, file.Filename))
			if err == nil {
				deleted = true
			} else {
				log.Errorf("Error deleting file ", err)
			}
		case "putio":
			id, err := strconv.ParseInt(file.Path, 10, 64)
			if err != nil {
				return scene
			}
			client := file.Volume.GetPutIOClient()
			err = client.Files.Delete(context.Background(), id)
			if err == nil {
				deleted = true
			} else {
				log.Errorf("Error deleting file ", err)
			}
		}

		if deleted {
			db.Delete(&file)
			if file.SceneID != 0 {
				scene.GetIfExistByPK(file.SceneID)
				scene.UpdateStatus()
			}
		}
	} else {
		log.Errorf("Error deleting file ", err)
	}
	return scene
}
