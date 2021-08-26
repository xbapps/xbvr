package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
	"golang.org/x/crypto/bcrypt"
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
	ID             uint                 `json:"id"`
	Title          string               `json:"title"`
	Authorized     uint                 `json:"authorized"`
	Description    string               `json:"description"`
	Paysite        DeoScenePaysite      `json:"paysite"`
	IsFavorite     bool                 `json:"isFavorite"`
	Is3D           bool                 `json:"is3d"`
	ThumbnailURL   string               `json:"thumbnailUrl"`
	RatingAvg      float64              `json:"rating_avg"`
	ScreenType     string               `json:"screenType"`
	StereoMode     string               `json:"stereoMode"`
	VideoLength    int                  `json:"videoLength"`
	VideoThumbnail string               `json:"videoThumbnail"`
	VideoPreview   string               `json:"videoPreview,omitempty"`
	Encodings      []DeoSceneEncoding   `json:"encodings"`
	Timestamps     []DeoSceneTimestamp  `json:"timeStamps"`
	Actors         []DeoSceneActor      `json:"actors"`
	Fleshlight     []DeoSceneScriptFile `json:"fleshlight,omitempty"`
	FullVideoReady bool                 `json:"fullVideoReady"`
	FullAccess     bool                 `json:"fullAccess"`
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

type DeoSceneScriptFile struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func isDeoAuthEnabled() bool {
	if config.Config.Interfaces.DeoVR.AuthEnabled &&
		config.Config.Interfaces.DeoVR.Username != "" &&
		config.Config.Interfaces.DeoVR.Password != "" {
		return true
	} else {
		return false
	}
}

func setDeoPlayerHost(req *restful.Request) {
	deoIP := req.Request.RemoteAddr
	lastColon := strings.LastIndex(deoIP, ":")
	if lastColon != -1 {
		deoIP = deoIP[:lastColon]
	}
	if deoIP != session.DeoPlayerHost {
		common.Log.Infof("DeoVR Player connecting from %v", deoIP)
		session.DeoPlayerHost = deoIP
		session.DeoRequestHost = "http://" + req.Request.Host
	}
}

func restfulAuthFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if isDeoAuthEnabled() {
		var authorized bool

		u, err := req.BodyParameter("login")
		if err != nil {
			authorized = false
		}

		p, err := req.BodyParameter("password")
		if err != nil {
			authorized = false
		}

		err = bcrypt.CompareHashAndPassword([]byte(config.Config.Interfaces.DeoVR.Password), []byte(p))
		if u == config.Config.Interfaces.DeoVR.Username && err == nil {
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

	ws.Route(ws.HEAD("").To(i.getDeoLibrary))

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
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	setDeoPlayerHost(req)

	dnt := ""
	if config.Config.Interfaces.DeoVR.RemoteEnabled {
		dnt = "?dnt=true"
	}

	db, _ := models.GetDB()
	defer db.Close()

	fileId, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		return
	}

	var file models.File
	db.Where(&models.File{ID: uint(fileId)}).First(&file)

	var height = file.VideoHeight
	var width = file.VideoWidth
	var sources []DeoSceneEncoding
	sources = append(sources, DeoSceneEncoding{
		Name: fmt.Sprintf("File 1/1 - %v", humanize.Bytes(uint64(file.Size))),
		VideoSources: []DeoSceneVideoSource{
			{
				Resolution: height,
				Height:     height,
				Width:      width,
				Size:       file.Size,
				URL:        fmt.Sprintf("%v/api/dms/file/%v%v", session.DeoRequestHost, file.ID, dnt),
			},
		},
	})

	deoScene := DeoScene{
		ID:           999900000 + file.ID,
		Authorized:   1,
		Description:  file.Filename,
		Title:        file.Filename,
		IsFavorite:   false,
		ThumbnailURL: session.DeoRequestHost + "/ui/images/blank.png",
		Is3D:         true,
		Encodings:    sources,
		VideoLength:  int(file.VideoDuration),
	}

	resp.WriteHeaderAndEntity(http.StatusOK, deoScene)
}

func (i DeoVRResource) getDeoScene(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	sceneID := req.PathParameter("scene-id")
	if sceneID == "" {
		return
	}

	setDeoPlayerHost(req)

	dnt := ""
	if config.Config.Interfaces.DeoVR.RemoteEnabled {
		dnt = "?dnt=true"
	}

	db, _ := models.GetDB()
	defer db.Close()

	var scene models.Scene
	err := db.Preload("Cast").
		Preload("Tags").
		Preload("Cuepoints").
		Where("id = ?", sceneID).First(&scene).Error
	if err != nil {
		log.Error(err)
		return
	}

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
	var videoFiles []models.File
	videoFiles, err = scene.GetVideoFiles()
	if err != nil {
		log.Error(err)
		return
	}

	for i, file := range videoFiles {
		var height = file.VideoHeight
		var width = file.VideoWidth

		sources = append(sources, DeoSceneEncoding{
			Name: fmt.Sprintf("File %v/%v %vp - %v", i+1, len(videoFiles), file.VideoHeight, humanize.Bytes(uint64(file.Size))),
			VideoSources: []DeoSceneVideoSource{
				{
					Resolution: height,
					Height:     height,
					Width:      width,
					Size:       file.Size,
					URL:        fmt.Sprintf("%v/api/dms/file/%v/%v%v", session.DeoRequestHost, file.ID, scene.GetFunscriptTitle(), dnt),
				},
			},
		})

		videoLength = file.VideoDuration
	}

	var deoScriptFiles []DeoSceneScriptFile
	var scriptFiles []models.File
	scriptFiles, err = scene.GetScriptFiles()
	if err != nil {
		log.Error(err)
		return
	}

	for _, file := range scriptFiles {
		deoScriptFiles = append(deoScriptFiles, DeoSceneScriptFile{
			Title: file.Filename,
			URL:   fmt.Sprintf("%v/api/dms/file/%v", session.DeoRequestHost, file.ID),
		})
	}

	var cuepoints []DeoSceneTimestamp
	for i := range scene.Cuepoints {
		cuepoints = append(cuepoints, DeoSceneTimestamp{
			TS:   uint(scene.Cuepoints[i].TimeStart),
			Name: scene.Cuepoints[i].Name,
		})
	}
	sort.Slice(cuepoints, func(i, j int) bool {
		return cuepoints[i].TS < cuepoints[j].TS
	})

	if videoFiles[0].VideoProjection == "mkx200" ||
		videoFiles[0].VideoProjection == "mkx220" ||
		videoFiles[0].VideoProjection == "vrca220" {
		stereoMode = "sbs"
		screenType = videoFiles[0].VideoProjection
	}

	if videoFiles[0].VideoProjection == "180_sbs" {
		stereoMode = "sbs"
		screenType = "dome"
	}

	if videoFiles[0].VideoProjection == "360_tb" {
		stereoMode = "tb"
		screenType = "sphere"
	}

	title := scene.Title
	thumbnailURL := session.DeoRequestHost + "/img/700x/" + strings.Replace(scene.CoverURL, "://", ":/", -1)

	if scene.IsScripted {
		title = scene.GetFunscriptTitle()
		if config.Config.Interfaces.DeoVR.RenderHeatmaps {
			thumbnailURL = session.DeoRequestHost + "/imghm/" + fmt.Sprint(scene.ID) + "/" + strings.Replace(scene.CoverURL, "://", ":/", -1)
		}
	}

	deoScene := DeoScene{
		ID:             scene.ID,
		Authorized:     1,
		Title:          title,
		Description:    scene.Synopsis,
		Actors:         actors,
		Paysite:        DeoScenePaysite{ID: 1, Name: scene.Site, Is3rdParty: true},
		IsFavorite:     scene.Favourite,
		RatingAvg:      scene.StarRating,
		FullVideoReady: true,
		FullAccess:     true,
		ThumbnailURL:   thumbnailURL,
		StereoMode:     stereoMode,
		Is3D:           true,
		ScreenType:     screenType,
		Encodings:      sources,
		VideoLength:    int(videoLength),
		Timestamps:     cuepoints,
		Fleshlight:     deoScriptFiles,
	}

	if scene.HasVideoPreview {
		deoScene.VideoPreview = fmt.Sprintf("%v/api/dms/preview/%v", session.DeoRequestHost, scene.SceneID)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, deoScene)
}

func (i DeoVRResource) getDeoLibrary(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	setDeoPlayerHost(req)

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
		Where("files.type = 'video'").
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

func scenesToDeoList(req *restful.Request, scenes []models.Scene) []DeoListItem {
	setDeoPlayerHost(req)

	list := make([]DeoListItem, 0)
	for i := range scenes {
		thumbnailURL := fmt.Sprintf("%v/img/700x/%v", session.DeoRequestHost, strings.Replace(scenes[i].CoverURL, "://", ":/", -1))

		if config.Config.Interfaces.DeoVR.RenderHeatmaps && scenes[i].IsScripted {
			thumbnailURL = fmt.Sprintf("%v/imghm/%d/%v", session.DeoRequestHost, scenes[i].ID, strings.Replace(scenes[i].CoverURL, "://", ":/", -1))
		}

		item := DeoListItem{
			Title:        scenes[i].Title,
			VideoLength:  scenes[i].Duration * 60,
			ThumbnailURL: thumbnailURL,
			VideoURL:     fmt.Sprintf("%v/deovr/%v", session.DeoRequestHost, scenes[i].ID),
		}
		list = append(list, item)
	}
	return list
}

func filesToDeoList(req *restful.Request, files []models.File) []DeoListItem {
	setDeoPlayerHost(req)

	dnt := ""
	if config.Config.Interfaces.DeoVR.RemoteEnabled {
		dnt = "?dnt=true"
	}

	list := make([]DeoListItem, 0)
	for i := range files {
		if files[i].Volume.Type == "local" {
			if !files[i].Volume.IsAvailable {
				continue
			}
		}
		item := DeoListItem{
			Title:        files[i].Filename,
			VideoLength:  int(files[i].VideoDuration),
			ThumbnailURL: session.DeoRequestHost + "/ui/images/blank.png",
			VideoURL:     fmt.Sprintf("%v/deovr/file/%v%v", session.DeoRequestHost, files[i].ID, dnt),
		}
		list = append(list, item)
	}
	return list
}
