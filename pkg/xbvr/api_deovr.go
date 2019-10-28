package xbvr

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/xbapps/xbvr/pkg/models"
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

type DeoSceneTimestamp struct {
	TS   uint   `json:"ts"`
	Name string `json:"name"`
}

type DeoScene struct {
	ID             uint                `json:"id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	IsFavorite     bool                `json:"isFavorite"`
	Is3D           bool                `json:"is3d"`
	ThumbnailURL   string              `json:"thumbnailUrl"`
	ScreenType     string              `json:"screenType"`
	StereoMode     string              `json:"stereoMode"`
	VideoLength    int                 `json:"videoLength"`
	VideoThumbnail string              `json:"videoThumbnail"`
	VideoPreview   string              `json:"videoPreview"`
	Encodings      []DeoSceneEncoding  `json:"encodings"`
	Timestamps     []DeoSceneTimestamp `json:"timeStamps"`
}

type DeoSceneActor struct {
	ID   uint   `json:"id"`
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

	ws.Route(ws.GET("").To(i.getDeoLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))

	ws.Route(ws.GET("/{scene-id}").To(i.getDeoScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))

	return ws
}

func (i DeoVRResource) getDeoScene(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var scene models.Scene
	db.Preload("Cast").
		Preload("Tags").
		Preload("Files").
		Preload("Cuepoints").
		Where(&models.Scene{SceneID: req.PathParameter("scene-id")}).First(&scene)

	baseURL := "http://" + req.Request.Host

	var stereoMode string
	var screenType string

	var actors []DeoSceneActor
	for i := range scene.Cast {
		actors = append(actors, DeoSceneActor{
			ID:   scene.Cast[i].ID,
			Name: scene.Cast[i].Name,
		})
	}

	var videoLength float64

	var sources []DeoSceneEncoding
	for i := range scene.Files {
		sources = append(sources, DeoSceneEncoding{
			Name: fmt.Sprintf("File %v/%v - %v", i+1, len(scene.Files), humanize.Bytes(uint64(scene.Files[i].Size))),
			VideoSources: []DeoSceneVideoSource{
				{
					Resolution: scene.Files[i].VideoHeight,
					Height:     scene.Files[i].VideoHeight,
					Width:      scene.Files[i].VideoWidth,
					Size:       scene.Files[i].Size,
					URL:        fmt.Sprintf("%v/api/dms/file/%v", baseURL, scene.Files[i].ID),
				},
			},
		})

		videoLength = scene.Files[i].VideoDuration
	}

	var cuepoints []DeoSceneTimestamp
	for i := range scene.Cuepoints {
		cuepoints = append(cuepoints, DeoSceneTimestamp{
			TS:   uint(scene.Cuepoints[i].TimeStart),
			Name: scene.Cuepoints[i].Name,
		})
	}

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
		Is3D:         true,
		ScreenType:   screenType,
		Encodings:    sources,
		VideoLength:  int(videoLength),
		Timestamps:   cuepoints,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, deoScene)
}

func (i DeoVRResource) getDeoLibrary(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var recent []models.Scene
	db.Model(&recent).
		Where("is_available = ?", true).
		Where("is_accessible = ?", true).
		Order("release_date desc").
		Find(&recent)

	var favourite []models.Scene
	db.Model(&favourite).
		Where("is_available = ?", true).
		Where("is_accessible = ?", true).
		Where("favourite = ?", true).
		Order("release_date desc").
		Find(&favourite)

	var watchlist []models.Scene
	db.Model(&watchlist).
		Where("is_available = ?", true).
		Where("is_accessible = ?", true).
		Where("watchlist = ?", true).
		Order("release_date desc").
		Find(&watchlist)

	lib := DeoLibrary{
		Authorized: 1,
		Scenes: []DeoListScenes{
			{
				Name: "Recent releases",
				List: scenesToDeoList(req, recent),
			},
			{
				Name: "Favourites",
				List: scenesToDeoList(req, favourite),
			},
			{
				Name: "Watchlist",
				List: scenesToDeoList(req, watchlist),
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

func scenesToDeoList(req *restful.Request, scenes []models.Scene) []DeoListItem {
	baseURL := "http://" + req.Request.Host

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
