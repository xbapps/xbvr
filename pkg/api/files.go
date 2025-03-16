package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
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

	ws.Route(ws.GET("/file/{file-id}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.File{}))

	ws.Route(ws.DELETE("/file/{file-id}").To(i.removeFile).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i FilesResource) getFile(req *restful.Request, resp *restful.Response) {
	var file models.File

	id, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		log.Error(err)
		return
	}

	_ = file.GetIfExistByPK(uint(id))

	switch file.Volume.Type {
	case "local":
		// Local file
		resp.Header().Set("Content-Disposition", "attachment; filename="+file.Filename)
		http.ServeFile(resp.ResponseWriter, req.Request, file.GetPath())
	case "putio":
		// Put.io file
		id, err := strconv.ParseInt(file.Path, 10, 64)
		if err != nil {
			return
		}
		client := file.Volume.GetPutIOClient()
		url, err := client.Files.URL(context.Background(), id, false)
		if err != nil {
			return
		}
		http.Redirect(resp.ResponseWriter, req.Request, url, http.StatusFound)
	case "debridlink":
		// Debrid-Link file
		// Create HTTP client with authorization header
		client := &http.Client{}

		// Extract the file ID from the path (which is stored as "displayPath||fileID")
		fileID := file.Path
		if strings.Contains(file.Path, "||") {
			parts := strings.Split(file.Path, "||")
			if len(parts) > 1 {
				fileID = parts[1]
			}
		}

		// Get file details to get download URL
		httpReq, err := http.NewRequest("GET", "https://debrid-link.com/api/v2/seedbox/list", nil)
		if err != nil {
			http.Error(resp.ResponseWriter, "Failed to create request", http.StatusInternalServerError)
			return
		}
		httpReq.Header.Add("Authorization", "Bearer "+file.Volume.Metadata)

		// Make request to get file list
		httpResp, err := client.Do(httpReq)
		if err != nil {
			http.Error(resp.ResponseWriter, "Failed to get file list", http.StatusInternalServerError)
			return
		}
		defer httpResp.Body.Close()

		// Parse response to find the file
		var filesResponse struct {
			Success bool `json:"success"`
			Value   []struct {
				Files []struct {
					ID          string `json:"id"`
					DownloadURL string `json:"downloadUrl"`
				} `json:"files"`
			} `json:"value"`
		}

		if err := json.NewDecoder(httpResp.Body).Decode(&filesResponse); err != nil {
			http.Error(resp.ResponseWriter, "Failed to parse response", http.StatusInternalServerError)
			return
		}

		// Find the file with matching ID
		var downloadURL string
		for _, torrent := range filesResponse.Value {
			for _, fileItem := range torrent.Files {
				if fileItem.ID == fileID {
					downloadURL = fileItem.DownloadURL
					break
				}
			}
			if downloadURL != "" {
				break
			}
		}

		if downloadURL == "" {
			http.Error(resp.ResponseWriter, "File not found", http.StatusNotFound)
			return
		}

		// Log the URL for debugging
		log.Infof("Proxying Debrid-Link URL: %s", downloadURL)

		// Create a new request to the download URL
		proxyReq, err := http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			http.Error(resp.ResponseWriter, "Failed to create proxy request", http.StatusInternalServerError)
			return
		}

		// Copy original request headers
		for header, values := range req.Request.Header {
			for _, value := range values {
				proxyReq.Header.Add(header, value)
			}
		}

		// If the request has a Range header, pass it through
		if rangeHeader := req.Request.Header.Get("Range"); rangeHeader != "" {
			proxyReq.Header.Set("Range", rangeHeader)
		}

		// Make the request to debrid-link
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(resp.ResponseWriter, "Failed to proxy request", http.StatusInternalServerError)
			return
		}
		defer proxyResp.Body.Close()

		// Set content type to video/mp4 if not specified
		contentType := proxyResp.Header.Get("Content-Type")
		if contentType == "" || contentType == "application/octet-stream" {
			contentType = "video/mp4"
		}
		resp.Header().Set("Content-Type", contentType)

		// Set content length if available
		if proxyResp.ContentLength > 0 {
			resp.Header().Set("Content-Length", fmt.Sprintf("%d", proxyResp.ContentLength))
		}

		// Enable byte range requests
		resp.Header().Set("Accept-Ranges", "bytes")

		// Copy other relevant headers
		for _, header := range []string{"Content-Range", "ETag", "Last-Modified"} {
			if value := proxyResp.Header.Get(header); value != "" {
				resp.Header().Set(header, value)
			}
		}

		// Set status code
		resp.WriteHeader(proxyResp.StatusCode)

		// Copy the body from the proxy response to our response
		_, err = io.Copy(resp.ResponseWriter, proxyResp.Body)
		if err != nil {
			// Check if it's a broken pipe error (client disconnected)
			if strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "connection reset by peer") {
				// This is normal when client stops the video or closes the page
				log.Debugf("Client disconnected during streaming: %v", err)
			} else {
				// Log other errors as errors
				log.Errorf("Error copying proxy response: %v", err)
			}
			return
		}
	}
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
		log.Infof("Deleting file %s", filepath.Join(file.Path, file.Filename))
		deleted := false
		switch file.Volume.Type {
		case "local":
			err := os.Remove(filepath.Join(file.Path, file.Filename))
			if err == nil || errors.Is(err, fs.ErrNotExist) {
				deleted = true
			} else {
				log.Errorf("error deleting file: %v", err)
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
				log.Errorf("error deleting file %v", err)
			}
		case "debridlink":
			// Extract the torrent ID from the file path (format: "displayPath||fileID")
			fileID := file.Path
			if strings.Contains(file.Path, "||") {
				parts := strings.Split(file.Path, "||")
				if len(parts) > 1 {
					fileID = parts[1]
				}
			}

			// Extract the torrent ID without the file suffix (e.g., "s5ng7xbxtitk4gg008socg8-1" -> "s5ng7xbxtitk4gg008socg8")
			torrentID := fileID
			if strings.Contains(fileID, "-") {
				parts := strings.Split(fileID, "-")
				torrentID = parts[0]
			}

			// Create HTTP client with authorization header
			client := &http.Client{}

			// Create DELETE request to remove the torrent
			deleteURL := fmt.Sprintf("https://debrid-link.com/api/v2/seedbox/%s/remove", torrentID)
			httpReq, err := http.NewRequest("DELETE", deleteURL, nil)
			if err != nil {
				log.Errorf("error creating DELETE request: %v", err)
				return scene
			}
			httpReq.Header.Add("Authorization", "Bearer "+file.Volume.Metadata)

			// Execute the request
			httpResp, err := client.Do(httpReq)
			if err != nil {
				log.Errorf("error deleting file from Debrid-Link: %v", err)
				return scene
			}
			defer httpResp.Body.Close()

			// Check response
			if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
				deleted = true
				log.Infof("Successfully deleted torrent %s from Debrid-Link", torrentID)
			} else {
				var errorResponse struct {
					Success bool   `json:"success"`
					Error   string `json:"error"`
				}

				if err := json.NewDecoder(httpResp.Body).Decode(&errorResponse); err == nil {
					log.Errorf("Debrid-Link API error: %s", errorResponse.Error)
				} else {
					log.Errorf("error deleting file from Debrid-Link, status code: %d", httpResp.StatusCode)
				}
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
		log.Errorf("error deleting file %v", err)
	}
	return scene
}
