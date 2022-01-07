package api

import (
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/models"
)

type CreateUpdatePlaylistRequest struct {
	Name         string `json:"name"`
	IsSmart      bool   `json:"is_smart"`
	IsDeoEnabled bool   `json:"is_deo_enabled"`
	SearchParams string `json:"search_params"`
}

type PlaylistResource struct{}

func (i PlaylistResource) WebService() *restful.WebService {
	tags := []string{"Playlist"}

	ws := new(restful.WebService)

	ws.Path("/api/playlist").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(i.listPlaylists).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes([]models.Playlist{}))

	ws.Route(ws.POST("").To(i.createPlaylist).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Playlist{}))

	ws.Route(ws.PUT("/{playlist-id}").To(i.updatePlaylist).
		Param(ws.PathParameter("playlist-id", "Playlist ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(models.Playlist{}))

	ws.Route(ws.DELETE("/{playlist-id}").To(i.removePlaylist).
		Param(ws.PathParameter("playlist-id", "Playlist ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i PlaylistResource) listPlaylists(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var playlists []models.Playlist
	db.Order("ordering asc").Find(&playlists)

	resp.WriteHeaderAndEntity(http.StatusOK, playlists)
}

func (i PlaylistResource) createPlaylist(req *restful.Request, resp *restful.Response) {
	var r CreateUpdatePlaylistRequest
	err := req.ReadEntity(&r)
	if err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	nv := models.Playlist{Name: r.Name, IsDeoEnabled: r.IsDeoEnabled, IsSmart: r.IsSmart, SearchParams: r.SearchParams}
	nv.Save()

	resp.WriteHeaderAndEntity(http.StatusOK, nv)
}

func (i PlaylistResource) updatePlaylist(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("playlist-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	var r CreateUpdatePlaylistRequest
	err = req.ReadEntity(&r)
	if err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	playlist := models.Playlist{}
	err = db.First(&playlist, id).Error

	if err == gorm.ErrRecordNotFound {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	playlist.Name = r.Name
	playlist.SearchParams = r.SearchParams
	playlist.IsDeoEnabled = r.IsDeoEnabled
	playlist.Save()

	resp.WriteHeaderAndEntity(http.StatusOK, playlist)
}

func (i PlaylistResource) removePlaylist(req *restful.Request, resp *restful.Response) {
	id, err := strconv.Atoi(req.PathParameter("playlist-id"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	playlist := models.Playlist{}
	err = db.First(&playlist, id).Error

	if err == gorm.ErrRecordNotFound {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	db.Where("id = ?", id).Delete(models.Playlist{})
	db.Delete(&playlist)

	resp.WriteHeader(http.StatusOK)
}
