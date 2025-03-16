package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
)

type DMSResource struct{}

func (i DMSResource) WebService() *restful.WebService {
	tags := []string{"DMS"}

	ws := new(restful.WebService)

	ws.Path("/api/dms").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/file/{file-id}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/file/{file-id}/{var:*}").To(i.getFile).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/heatmap/{file-id}").To(i.getHeatmap).
		Param(ws.PathParameter("file-id", "File ID").DataType("int")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/preview/{scene-id}").To(i.getPreview).
		Param(ws.PathParameter("scene-id", "Scene ID")).
		ContentEncodingEnabled(false).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i DMSResource) getPreview(req *restful.Request, resp *restful.Response) {
	sceneID := req.PathParameter("scene-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", sceneID)))
}

func (i DMSResource) getHeatmap(req *restful.Request, resp *restful.Response) {
	fileID := req.PathParameter("file-id")
	http.ServeFile(resp.ResponseWriter, req.Request, filepath.Join(common.ScriptHeatmapDir, fmt.Sprintf("heatmap-%v.png", fileID)))
}

func (i DMSResource) getFile(req *restful.Request, resp *restful.Response) {
	doNotTrack := req.QueryParameter("dnt")
	id, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if scene exist
	db, _ := models.GetDB()
	defer db.Close()

	f := models.File{}
	err = db.Preload("Volume").First(&f, id).Error

	switch f.Volume.Type {
	case "local":
		// Track current session
		setDeoPlayerHost(req)
		session.TrackSessionFromFile(f, doNotTrack)

		if err == gorm.ErrRecordNotFound {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		ctx := req.Request.Context()
		http.ServeFile(resp.ResponseWriter, req.Request, f.GetPath())
		select {
		case <-ctx.Done():
			session.FinishTrackingFromFile(doNotTrack)
			return
		default:
		}
	case "putio":
		id, err := strconv.ParseInt(f.Path, 10, 64)
		if err != nil {
			return
		}
		client := f.Volume.GetPutIOClient()
		url, err := client.Files.URL(context.Background(), id, false)
		if err != nil {
			return
		}
		http.Redirect(resp.ResponseWriter, req.Request, url, http.StatusFound)
	case "debridlink":
		// Create HTTP client with authorization header
		client := &http.Client{}

		// Extract the file ID from the path (which is stored as "displayPath||fileID")
		fileID := f.Path
		if strings.Contains(f.Path, "||") {
			parts := strings.Split(f.Path, "||")
			if len(parts) > 1 {
				fileID = parts[1]
			}
		}

		// Get file details to get download URL
		httpReq, err := http.NewRequest("GET", "https://debrid-link.com/api/v2/seedbox/list", nil)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		httpReq.Header.Add("Authorization", "Bearer "+f.Volume.Metadata)

		// Make request to get file list
		httpResp, err := client.Do(httpReq)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
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
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Find the file with matching ID
		var downloadURL string
		for _, torrent := range filesResponse.Value {
			for _, file := range torrent.Files {
				if file.ID == fileID {
					downloadURL = file.DownloadURL
					break
				}
			}
			if downloadURL != "" {
				break
			}
		}

		if downloadURL == "" {
			resp.WriteHeader(http.StatusNotFound)
			return
		}

		// Log the URL for debugging
		log.Infof("Proxying Debrid-Link URL: %s", downloadURL)

		// Create a new request to the download URL
		proxyReq, err := http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
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
			resp.WriteHeader(http.StatusInternalServerError)
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
