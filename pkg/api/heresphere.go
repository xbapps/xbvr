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
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

type HeresphereLibrary struct {
	Access  int                    `json:"access"`
	Banner  HeresphereBanner       `json:"banner"`
	Library []HeresphereListScenes `json:"library"`
}

type HeresphereBanner struct {
	Image string `json:"image"`
	Link  string `json:"link"`
}

type HeresphereListScenes struct {
	Name string   `json:"name"`
	List []string `json:"list"`
}

type HeresphereVideo struct {
	Access               int                `json:"access"`
	Title                string             `json:"title"`
	Description          string             `json:"description"`
	ThumbnailImage       string             `json:"thumbnailImage"`
	ThumbnailVideo       string             `json:"thumbnailVideo,omitempty"`
	DateReleased         string             `json:"dateReleased"`
	DateAdded            string             `json:"dateAdded"`
	DurationMilliseconds uint               `json:"duration"`
	Rating               float64            `json:"rating"`
	IsFavorite           bool               `json:"isFavorite"`
	Projection           string             `json:"projection"`
	Stereo               string             `json:"stereo"`
	FOV                  float64            `json:"fov"`
	Lens                 string             `json:"lens"`
	HspUrl               string             `json:"hsp,omitempty"`
	Scripts              []HeresphereScript `json:"scripts,omitempty"`
	Tags                 []HeresphereTag    `json:"tags"`
	Media                []HeresphereMedia  `json:"media"`
}

type HeresphereScript struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type HeresphereTag struct {
	Name              string `json:"name"`
	StartMilliseconds int    `json:"start,omitempty"`
	EndMilliseconds   int    `json:"end,omitempty"`
	Track             *int   `json:"track,omitempty"`
}

type HeresphereMedia struct {
	Name    string             `json:"name"`
	Sources []HeresphereSource `json:"sources"`
}

type HeresphereSource struct {
	Resolution int    `json:"resolution"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
	Size       int64  `json:"size"`
	URL        string `json:"url"`
}

type HereSphereAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func HeresphereAuthFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
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
			unauthLib := HeresphereLibrary{
				Access: -1,
				Library: []HeresphereListScenes{
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

func HeresphereResponseFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	resp.AddHeader("HereSphere-JSON-Version", "1")
	chain.ProcessFilter(req, resp)
}

type HeresphereResource struct{}

func (i HeresphereResource) WebService() *restful.WebService {
	tags := []string{"HereSphere"}

	ws := new(restful.WebService)

	ws.Path("/heresphere/").
		Filter(HeresphereResponseFilter).
		Consumes(restful.MIME_JSON, "application/x-www-form-urlencoded").
		Produces(restful.MIME_JSON)

	ws.Route(ws.HEAD("").To(i.getHeresphereLibrary))

	ws.Route(ws.GET("").Filter(restfulAuthFilter).To(i.getHeresphereLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))
	ws.Route(ws.POST("").Filter(restfulAuthFilter).To(i.getHeresphereLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))

	ws.Route(ws.GET("/{scene-id}").Filter(restfulAuthFilter).To(i.getHeresphereScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	ws.Route(ws.POST("/{scene-id}").Filter(restfulAuthFilter).To(i.getHeresphereScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))

	ws.Route(ws.GET("/file/{file-id}").Filter(restfulAuthFilter).To(i.getHeresphereFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	ws.Route(ws.POST("file/{file-id}").Filter(restfulAuthFilter).To(i.getHeresphereFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	return ws
}

func (i HeresphereResource) getHeresphereFile(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	fileId, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil {
		return
	}

	var file models.File
	db.Where(&models.File{ID: uint(fileId)}).First(&file)

	var resolution = file.VideoHeight
	var height = file.VideoHeight
	var width = file.VideoWidth
	if file.VideoProjection == "360_tb" {
		resolution = resolution / 2
	}

	var media []HeresphereMedia
	media = append(media, HeresphereMedia{
		Name: fmt.Sprintf("File 1/1 %vp - %v", resolution, humanize.Bytes(uint64(file.Size))),
		Sources: []HeresphereSource{
			{
				Resolution: resolution,
				Height:     height,
				Width:      width,
				Size:       file.Size,
				URL:        fmt.Sprintf("http://%v/api/dms/file/%v/%v", req.Request.Host, file.ID, file.Filename),
			},
		},
	})

	video := HeresphereVideo{
		Access:               1,
		Title:                file.Filename,
		Description:          file.Filename,
		ThumbnailImage:       "http://" + req.Request.Host + "/ui/images/blank.png",
		DateReleased:         file.CreatedTime.Format("2006-01-02"),
		DateAdded:            file.CreatedTime.Format("2006-01-02"),
		DurationMilliseconds: uint(file.VideoDuration * 1000),
		Media:                media,
	}

	resp.WriteHeaderAndEntity(http.StatusOK, video)
}

func (i HeresphereResource) getHeresphereScene(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	sceneID := req.PathParameter("scene-id")
	if sceneID == "" {
		return
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

	var media []HeresphereMedia

	var videoFiles []models.File
	videoFiles, err = scene.GetVideoFiles()
	if err != nil {
		log.Error(err)
		return
	}
	videoLength := float64(scene.Duration)

	for i, file := range videoFiles {
		var height = file.VideoHeight
		var width = file.VideoWidth
		var resolution = file.VideoHeight

		if file.VideoProjection == "360_tb" {
			resolution = resolution / 2
		}

		var mediafile = HeresphereMedia{
			Name: fmt.Sprintf("File %v/%v %vp - %v", i+1, len(videoFiles), file.VideoHeight, humanize.Bytes(uint64(file.Size))),
			Sources: []HeresphereSource{
				{
					Resolution: resolution,
					Height:     height,
					Width:      width,
					Size:       file.Size,
					URL:        fmt.Sprintf("http://%v/api/dms/file/%v/%v", req.Request.Host, file.ID, scene.GetFunscriptTitle()),
				},
			},
		}

		media = append(media, mediafile)
		videoLength = file.VideoDuration
	}

	var tags []HeresphereTag

	cuepoints := scene.Cuepoints
	sort.Slice(cuepoints, func(i, j int) bool {
		return cuepoints[i].TimeStart < cuepoints[j].TimeStart
	})

	track := 0
	end := 0
	for i := range cuepoints {
		start := int(cuepoints[i].TimeStart * 1000)
		if i+1 < len(cuepoints) {
			end = int(cuepoints[i+1].TimeStart*1000) - 1
		} else if int(videoLength*1000) > start {
			end = int(videoLength * 1000)
		} else {
			end = start + 1000
		}
		tags = append(tags, HeresphereTag{
			Name:              scene.Cuepoints[i].Name,
			StartMilliseconds: start,
			EndMilliseconds:   end,
			Track:             &track,
		})
	}

	tags = append(tags, HeresphereTag{
		Name: "Studio:" + scene.Site,
	})

	for i := range scene.Cast {
		tags = append(tags, HeresphereTag{
			Name: "Talent:" + scene.Cast[i].Name,
		})
	}

	for i := range scene.Tags {
		tags = append(tags, HeresphereTag{
			Name: "Tag:" + scene.Tags[i].Name,
		})
	}

	var heresphereScriptFiles []HeresphereScript
	var scriptFiles []models.File
	scriptFiles, err = scene.GetScriptFiles()
	if err != nil {
		log.Error(err)
		return
	}

	for _, file := range scriptFiles {
		heresphereScriptFiles = append(heresphereScriptFiles, HeresphereScript{
			Name: file.Filename,
			URL:  fmt.Sprintf("http://%v/api/dms/file/%v", req.Request.Host, file.ID),
		})
	}

	hspUrl := ""
	var hspFiles []models.File
	hspFiles, err = scene.GetHSPFiles()
	if err != nil {
		log.Error(err)
		return
	}

	if len(hspFiles) > 0 {
		hspUrl = fmt.Sprintf("http://%v/api/dms/file/%v", req.Request.Host, hspFiles[0].ID)
	}

	var projection string = "equirectangular"
	var stereo string = "sbs"
	var fov = 180.0
	var lens = "Linear"

	switch videoFiles[0].VideoProjection {
	case "flat":
		projection = "perspective"
		stereo = "mono"

	case "180_mono":
		projection = "equirectangular"
		stereo = "mono"

	case "360_mono":
		projection = "equirectangular360"
		stereo = "mono"

	case "180_sbs":
		projection = "equirectangular"

	case "360_tb":
		projection = "equirectangular360"
		stereo = "tb"

	case "mkx200":
		projection = "fisheye"
		fov = 200.0
		lens = "MKX200"

	case "mkx220":
		projection = "fisheye"
		fov = 220.0
		lens = "MKX220"

	case "vrca220":
		projection = "fisheye"
		fov = 220.0
		lens = "VRCA220"

	case "rf52":
		projection = "fisheye"
		fov = 190.0

	case "fisheye190":
		projection = "fisheye"
		fov = 190.0

	case "fisheye":
		projection = "fisheye"
		fov = 180.0
	}

	title := scene.Title
	thumbnailURL := "http://" + req.Request.Host + "/img/700x/" + strings.Replace(scene.CoverURL, "://", ":/", -1)

	if scene.IsScripted {
		if config.Config.Interfaces.DeoVR.RenderHeatmaps {
			thumbnailURL = "http://" + req.Request.Host + "/imghm/" + fmt.Sprint(scene.ID) + "/" + strings.Replace(scene.CoverURL, "://", ":/", -1)
		}
	}

	video := HeresphereVideo{
		Access:               1,
		Title:                title,
		Description:          scene.Synopsis,
		ThumbnailImage:       thumbnailURL,
		DateReleased:         scene.ReleaseDate.Format("2006-01-02"),
		DateAdded:            scene.AddedDate.Format("2006-01-02"),
		DurationMilliseconds: uint(videoLength * 1000),
		Rating:               scene.StarRating,
		IsFavorite:           scene.Favourite,
		Projection:           projection,
		Stereo:               stereo,
		FOV:                  fov,
		Lens:                 lens,
		HspUrl:               hspUrl,
		Scripts:              heresphereScriptFiles,
		Tags:                 tags,
		Media:                media,
	}

	if scene.HasVideoPreview {
		video.ThumbnailVideo = fmt.Sprintf("http://%v/api/dms/preview/%v", req.Request.Host, scene.SceneID)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, video)
}

func (i HeresphereResource) getHeresphereLibrary(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var sceneLists []HeresphereListScenes

	var savedPlaylists []models.Playlist
	db.Where("is_deo_enabled = ?", true).Order("ordering asc").Find(&savedPlaylists)

	for i := range savedPlaylists {
		var r models.RequestSceneList

		if err := json.Unmarshal([]byte(savedPlaylists[i].SearchParams), &r); err == nil {
			r.IsAccessible = optional.NewBool(true)
			r.IsAvailable = optional.NewBool(true)
			r.Limit = optional.NewInt(10000)

			q := models.QueryScenes(r, false)

			list := make([]string, 0)
			for i := range q.Scenes {
				url := fmt.Sprintf("http://%v/heresphere/%v", req.Request.Host, q.Scenes[i].ID)
				list = append(list, url)
			}

			sceneLists = append(sceneLists, HeresphereListScenes{
				Name: savedPlaylists[i].Name,
				List: list,
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

	list := make([]string, 0)
	for i := range unmatched {
		url := fmt.Sprintf("http://%v/heresphere/file/%v", req.Request.Host, unmatched[i].ID)
		list = append(list, url)
	}

	sceneLists = append(sceneLists, HeresphereListScenes{
		Name: "Unmatched",
		List: list,
	})

	resp.WriteHeaderAndEntity(http.StatusOK, HeresphereLibrary{
		Access:  1,
		Library: sceneLists,
	})
}
