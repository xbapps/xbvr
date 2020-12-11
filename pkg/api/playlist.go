package api

import (
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type NewPlaylistRequest struct {
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
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("").To(i.createPlaylist).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.PUT("/{id}").To(i.updatePlaylist).
		Param(ws.PathParameter("playlist-id", "Playlist ID").DataType("int")).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.DELETE("/{id}").To(i.removePlaylist).
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
	var r NewPlaylistRequest
	err := req.ReadEntity(&r)
	if err != nil {
		APIError(req, resp, http.StatusInternalServerError, err)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	nv := models.Playlist{Name: r.Name, IsDeoEnabled: r.IsDeoEnabled, IsSmart: r.IsSmart, SearchParams: r.SearchParams}
	nv.Save()

	resp.WriteHeader(http.StatusOK)
}

func (i PlaylistResource) updatePlaylist(req *restful.Request, resp *restful.Response) {
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

	playlist.IsDeoEnabled = !playlist.IsDeoEnabled
	playlist.Save()

	var playlists []models.Site
	db.Find(&playlists)
	resp.WriteHeaderAndEntity(http.StatusOK, playlists)
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

	// Inform UI about state change
	common.PublishWS("state.change.optionsStorage", nil)

	resp.WriteHeader(http.StatusOK)
}
