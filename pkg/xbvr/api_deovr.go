package xbvr

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/models"
)

type DeoLibrary struct {
	Scenes     []DeoListScenes `json:"scenes"`
	Authorized string          `json:"authorized"`
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
	Authorized     uint                `json:"authorized"`
	Description    string              `json:"description"`
	Paysite        DeoScenePaysite     `json:"paysite"`
	IsFavorite     bool                `json:"isFavorite"`
	Is3D           bool                `json:"is3d"`
	ThumbnailURL   string              `json:"thumbnailUrl"`
	RatingAvg      float64             `json:"rating_avg"`
	ScreenType     string              `json:"screenType"`
	StereoMode     string              `json:"stereoMode"`
	VideoLength    int                 `json:"videoLength"`
	VideoThumbnail string              `json:"videoThumbnail"`
	VideoPreview   string              `json:"videoPreview,omitempty"`
	Encodings      []DeoSceneEncoding  `json:"encodings"`
	Timestamps     []DeoSceneTimestamp `json:"timeStamps"`
	Actors         []DeoSceneActor     `json:"actors"`
	Categories     []DeoSceneCategory  `json:"categories"`
	FullVideoReady bool                `json:"fullVideoReady"`
	FullAccess     bool                `json:"fullAccess"`
}

type DeoSceneActor struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type DeoSceneCategory struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type DeoScenePaysite struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Is3rdParty bool   `json:"is3rdParty"`
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

func deoAuthEnabled() bool {
	if DEOPASSWORD != "" && DEOUSER != "" {
		return true
	} else {
		return false
	}
}

func restfulAuthFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if deoAuthEnabled() {
		var authorized bool

		u, err := req.BodyParameter("login")
		if err != nil {
			authorized = false
		}

		p, err := req.BodyParameter("password")
		if err != nil {
			authorized = false
		}

		if u == DEOUSER && p == DEOPASSWORD {
			authorized = true
		}

		if !authorized {
			unauthLib := DeoLibrary{
				Authorized: "-1",
				Scenes: []DeoListScenes{
					{
						Name: "Login Required",
						List: nil,
					},
				},
			}
			resp.WriteHeaderAndEntity(http.StatusOK, unauthLib)
			return
		}
	}
	chain.ProcessFilter(req, resp)
}

type DeoVRResource struct{}

func (i DeoVRResource) WebService() *restful.WebService {
	tags := []string{"DeoVR"}

	ws := new(restful.WebService)

	ws.Path("/deovr/").
		Consumes(restful.MIME_JSON, "application/x-www-form-urlencoded").
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("").Filter(restfulAuthFilter).To(i.getDeoLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))
	ws.Route(ws.POST("").Filter(restfulAuthFilter).To(i.getDeoLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))

	ws.Route(ws.GET("/{scene-id}").Filter(restfulAuthFilter).To(i.getDeoScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	ws.Route(ws.POST("/{scene-id}").Filter(restfulAuthFilter).To(i.getDeoScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))

	ws.Route(ws.GET("/file/{file-id}").Filter(restfulAuthFilter).To(i.getDeoFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	ws.Route(ws.POST("file/{file-id}").Filter(restfulAuthFilter).To(i.getDeoFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))

	return ws
}

func (i DeoVRResource) getDeoFile(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	fileId, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		return
	}

	baseURL := "http://" + req.Request.Host

	var file models.File
	db.Where(&models.File{ID: uint(fileId)}).First(&file)

	//TODO: remove temporary workaround, once DeoVR doesn't block hi-res videos anymore
	var height = file.VideoHeight
	var width = file.VideoWidth
	if height > 2160 {
		height = height / 10
		width = width / 10
	}

	var sources []DeoSceneEncoding
	sources = append(sources, DeoSceneEncoding{
		Name: fmt.Sprintf("File 1/1 - %v", humanize.Bytes(uint64(file.Size))),
		VideoSources: []DeoSceneVideoSource{
			{
				Resolution: height,
				Height:     height,
				Width:      width,
				Size:       file.Size,
				URL:        fmt.Sprintf("%v/api/dms/file/%v", baseURL, file.ID),
			},
		},
	})

	deoScene := DeoScene{
		ID:           999900000 + file.ID,
		Authorized:   1,
		Description:  file.Filename,
		Title:        file.Filename,
		IsFavorite:   false,
		ThumbnailURL: baseURL + "/ui/images/blank.png",
		Is3D:         true,
		Encodings:    sources,
		VideoLength:  int(file.VideoDuration),
	}

	resp.WriteHeaderAndEntity(http.StatusOK, deoScene)
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

	var categories []DeoSceneCategory
	for i := range scene.Tags {
		categories = append(categories, DeoSceneCategory{
			ID:   scene.Tags[i].ID,
			Name: scene.Tags[i].Name,
		})
	}

	var videoLength float64

	var sources []DeoSceneEncoding
	for i := range scene.Files {
		//TODO: remove temporary workaround, once DeoVR doesn't block hi-res videos anymore
		var height = scene.Files[i].VideoHeight
		var width = scene.Files[i].VideoWidth
		if height > 2160 {
			height = height / 10
			width = width / 10
		}
		sources = append(sources, DeoSceneEncoding{
			Name: fmt.Sprintf("File %v/%v %vp - %v", i+1, len(scene.Files), scene.Files[i].VideoHeight, humanize.Bytes(uint64(scene.Files[i].Size))),
			VideoSources: []DeoSceneVideoSource{
				{
					Resolution: height,
					Height:     height,
					Width:      width,
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
		ID:             scene.ID,
		Authorized:     1,
		Title:          scene.Title,
		Description:    scene.Synopsis,
		Actors:         actors,
		Categories:     categories,
		Paysite:        DeoScenePaysite{ID: 1, Name: scene.Site, Is3rdParty: true},
		IsFavorite:     scene.Favourite,
		RatingAvg:      scene.StarRating,
		FullVideoReady: true,
		FullAccess:     true,
		ThumbnailURL:   baseURL + "/img/700x/" + strings.Replace(scene.CoverURL, "://", ":/", -1),
		StereoMode:     stereoMode,
		Is3D:           true,
		ScreenType:     screenType,
		Encodings:      sources,
		VideoLength:    int(videoLength),
		Timestamps:     cuepoints,
	}

	if scene.HasVideoPreview {
		deoScene.VideoPreview = fmt.Sprintf("%v/api/dms/preview/%v", baseURL, scene.SceneID)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, deoScene)
}

func (i DeoVRResource) getDeoLibrary(req *restful.Request, resp *restful.Response) {
	db, _ := models.GetDB()
	defer db.Close()

	var sceneLists []DeoListScenes

	var savedPlaylists []models.Playlist
	db.Where("is_deo_enabled = ?", true).Order("ordering asc").Find(&savedPlaylists)

	for i := range savedPlaylists {
		var r models.RequestSceneList

		if err := json.Unmarshal([]byte(savedPlaylists[i].SearchParams), &r); err == nil {
			r.IsAccessible = optional.NewBool(true)
			r.IsAvailable = optional.NewBool(true)
			r.Limit = optional.NewInt(10000)

			q := models.QueryScenes(r, false)
			sceneLists = append(sceneLists, DeoListScenes{
				Name: savedPlaylists[i].Name,
				List: scenesToDeoList(req, q.Scenes),
			})
		}
	}

	// Add unmatched files at the end
	var unmatched []models.File
	db.Model(&unmatched).
		Preload("Volume").
		Where("files.scene_id = 0").
		Order("created_time desc").
		Find(&unmatched)

	sceneLists = append(sceneLists, DeoListScenes{
		Name: "Unmatched",
		List: filesToDeoList(req, unmatched),
	})

	resp.WriteHeaderAndEntity(http.StatusOK, DeoLibrary{
		Authorized: "1",
		Scenes:     sceneLists,
	})
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

func filesToDeoList(req *restful.Request, files []models.File) []DeoListItem {
	baseURL := "http://" + req.Request.Host

	var list []DeoListItem
	for i := range files {
		if files[i].Volume.Type == "local" {
			if !files[i].Volume.IsAvailable {
				continue
			}
		}
		item := DeoListItem{
			Title:        files[i].Filename,
			VideoLength:  int(files[i].VideoDuration),
			ThumbnailURL: baseURL + "/ui/images/blank.png",
			VideoURL:     baseURL + "/deovr/file/" + fmt.Sprint(files[i].ID),
		}
		list = append(list, item)
	}
	return list
}
