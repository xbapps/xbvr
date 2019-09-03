package xbvr

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
)

type DeoLibrary struct {
	Scenes     []DeoListScenes `json:"scenes"`
	Authorized int             `json:"authorized"`
}

type DeoListScenes struct {
	Name string        `json:"name"`
	List []DeoListItem `json:"list"`
}

type DeoListItem struct {
	Title        string `json:"title"`
	VideoLength  int    `json:"videoLength"`
	ThumbnailURL string `json:"thumbnailUrl"`
	VideoURL     string `json:"video_url"`
}

type DeoScene struct {
	ID             uint               `json:"id"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	IsFavorite     bool               `json:"isFavorite"`
	ThumbnailURL   string             `json:"thumbnailUrl"`
	ScreenType     string             `json:"screenType"`
	StereoMode     string             `json:"stereoMode"`
	VideoLength    int                `json:"videoLength"`
	VideoThumbnail string             `json:"videoThumbnail"`
	VideoPreview   string             `json:"videoPreview"`
	Encodings      []DeoSceneEncoding `json:"encodings"`
}

type DeoSceneActor struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DeoSceneEncoding struct {
	Name         string                `json:"name"`
	VideoSources []DeoSceneVideoSource `json:"videoSources"`
}

type DeoSceneVideoSource struct {
	Resolution int    `json:"resolution"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
	Size       int64  `json:"size"`
	URL        string `json:"url"`
}

type DeoVRResource struct{}

func (i DeoVRResource) WebService() *restful.WebService {
	tags := []string{"DeoVR"}

	ws := new(restful.WebService)

	ws.Path("/deovr/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").To(i.getScenes).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))

	ws.Route(ws.GET("/{scene-id}").To(i.getScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))

	return ws
}

func (i DeoVRResource) getScene(req *restful.Request, resp *restful.Response) {
	db, _ := GetDB()
	defer db.Close()

	log.Info(req.PathParameter("scene-id"))

	var scene Scene
	db.Preload("Cast").
		Preload("Tags").
		Preload("Files").
		Where(&Scene{SceneID: req.PathParameter("scene-id")}).First(&scene)

	baseURL := "http://" + getBaseURL() + ":9999"

	var stereoMode string
	var screenType string

	if scene.Files[0].VideoProjection == "180_sbs" {
		stereoMode = "sbs"
		screenType = "dome"
	}

	if scene.Files[0].VideoProjection == "360_tb" {
		stereoMode = "tb"
		screenType = "sphere"
	}

	deoScene := DeoScene{
		ID:           scene.ID,
		Title:        scene.Title,
		Description:  scene.Synopsis,
		IsFavorite:   scene.Favourite,
		ThumbnailURL: baseURL + "/img/700x/" + strings.Replace(scene.CoverURL, "://", ":/", -1),
		StereoMode:   stereoMode,
		ScreenType:   screenType,
		Encodings: []DeoSceneEncoding{
			{
				Name: "default",
				VideoSources: []DeoSceneVideoSource{
					{
						Resolution: scene.Files[0].VideoHeight,
						Height:     scene.Files[0].VideoHeight,
						Width:      scene.Files[0].VideoWidth,
						Size:       scene.Files[0].Size,
						URL:        fmt.Sprintf("%v/api/dms/file/%v", baseURL, scene.Files[0].ID),
					},
				},
			},
		},
	}

	resp.WriteHeaderAndEntity(http.StatusOK, deoScene)
}

func (i DeoVRResource) getScenes(req *restful.Request, resp *restful.Response) {
	var limit = 100
	var offset = 0

	db, _ := GetDB()
	defer db.Close()

	var recent []Scene
	db.Model(&recent).
		Where("is_available = ?", true).
		Where("is_accessible = ?", true).
		Order("release_date desc").
		Limit(limit).
		Offset(offset).
		Find(&recent)

	var favourite []Scene
	db.Model(&favourite).
		Where("is_available = ?", true).
		Where("is_accessible = ?", true).
		Where("favourite = ?", true).
		Order("release_date desc").
		Limit(limit).
		Offset(offset).
		Find(&favourite)

	var watchlist []Scene
	db.Model(&watchlist).
		Where("is_available = ?", true).
		Where("is_accessible = ?", true).
		Where("watchlist = ?", true).
		Order("release_date desc").
		Limit(limit).
		Offset(offset).
		Find(&watchlist)

	lib := DeoLibrary{
		Authorized: 1,
		Scenes: []DeoListScenes{
			{
				Name: "Recent releases",
				List: scenesToDeoList(recent),
			},
			{
				Name: "Favourites",
				List: scenesToDeoList(favourite),
			},
			{
				Name: "Watchlist",
				List: scenesToDeoList(watchlist),
			},
		},
	}

	resp.WriteHeaderAndEntity(http.StatusOK, lib)
}

func getBaseURL() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return hostname
			}
			return string(ip)
		}
	}
	return hostname
}

func scenesToDeoList(scenes []Scene) []DeoListItem {
	baseURL := "http://" + getBaseURL() + ":9999"

	var list []DeoListItem
	for i := range scenes {
		item := DeoListItem{
			Title:        scenes[i].Title,
			VideoLength:  scenes[i].Duration * 60,
			ThumbnailURL: baseURL + "/img/700x/" + strings.Replace(scenes[i].CoverURL, "://", ":/", -1),
			VideoURL:     baseURL + "/deovr/" + scenes[i].SceneID,
		}
		list = append(list, item)
	}
	return list
}
