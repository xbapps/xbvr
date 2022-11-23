package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/tasks"
	"golang.org/x/crypto/bcrypt"
)

type HeresphereLibrary struct {
	Access  int                    `json:"access"`
	Library []HeresphereListScenes `json:"library"`
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
	Rating               float64            `json:"rating,omitempty"`
	IsFavorite           bool               `json:"isFavorite"`
	Projection           string             `json:"projection"`
	Stereo               string             `json:"stereo"`
	FOV                  float64            `json:"fov"`
	Lens                 string             `json:"lens"`
	HspUrl               string             `json:"hsp,omitempty"`
	Scripts              []HeresphereScript `json:"scripts,omitempty"`
	Tags                 []HeresphereTag    `json:"tags,omitempty"`
	Media                []HeresphereMedia  `json:"media"`
	WriteFavorite        bool               `json:"writeFavorite"`
	WriteRating          bool               `json:"writeRating"`
	WriteTags            bool               `json:"writeTags"`
	WriteHSP             bool               `json:"writeHSP"`
}

type HeresphereScript struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type HeresphereTag struct {
	Name              string  `json:"name"`
	StartMilliseconds float64 `json:"start,omitempty"`
	EndMilliseconds   float64 `json:"end,omitempty"`
	Track             *int    `json:"track,omitempty"`
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
	Username    string           `json:"username"`
	Password    string           `json:"password"`
	Rating      *float64         `json:"rating"`
	IsFavorite  *bool            `json:"isFavorite"`
	Hsp         *string          `json:"hsp"`
	Tags        *[]HeresphereTag `json:"tags"`
	DeleteFiles *bool            `json:"deleteFile"`
}

var RequestBody []byte

func HeresphereAuthFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	RequestBody, _ = ioutil.ReadAll(req.Request.Body)
	if isDeoAuthEnabled() {
		var authorized bool
		var requestData HereSphereAuthRequest

		if err := json.Unmarshal(RequestBody, &requestData); err != nil {
			authorized = false
		} else {
			err := bcrypt.CompareHashAndPassword([]byte(config.Config.Interfaces.DeoVR.Password), []byte(requestData.Password))
			if requestData.Username == config.Config.Interfaces.DeoVR.Username && err == nil {
				authorized = true
			}
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

	ws.Route(ws.GET("").Filter(HeresphereAuthFilter).To(i.getHeresphereLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))
	ws.Route(ws.POST("").Filter(HeresphereAuthFilter).To(i.getHeresphereLibrary).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoLibrary{}))

	ws.Route(ws.GET("/{scene-id}").Filter(HeresphereAuthFilter).To(i.getHeresphereScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	ws.Route(ws.POST("/{scene-id}").Filter(HeresphereAuthFilter).To(i.getHeresphereScene).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))

	ws.Route(ws.GET("/file/{file-id}").Filter(HeresphereAuthFilter).To(i.getHeresphereFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	ws.Route(ws.POST("file/{file-id}").Filter(HeresphereAuthFilter).To(i.getHeresphereFile).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(DeoScene{}))
	return ws
}

func (i HeresphereResource) getHeresphereFile(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	var requestData HereSphereAuthRequest
	if err := json.Unmarshal(RequestBody, &requestData); err != nil {
		log.Warnf("Error decoding heresphere api POST request: %v %s", err, req.Request.RequestURI)
	}

	dnt := ""
	if !config.Config.Interfaces.DeoVR.TrackWatchTime {
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
				URL:        fmt.Sprintf("http://%v/api/dms/file/%v/%v/%v", req.Request.Host, file.ID, file.Filename, dnt),
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
	if requestData.DeleteFiles != nil && config.Config.Interfaces.Heresphere.AllowFileDeletes {
		removeFileByFileId(file.ID)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, video)
}

func (i HeresphereResource) getHeresphereScene(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}

	var requestData HereSphereAuthRequest

	if err := json.Unmarshal(RequestBody, &requestData); err != nil {
		log.Warnf("Error decoding heresphere api POST request: %v %s", err, req.Request.RequestURI)
	}

	sceneID := req.PathParameter("scene-id")
	if sceneID == "" {
		return
	}

	dnt := ""
	if !config.Config.Interfaces.DeoVR.TrackWatchTime {
		dnt = "?dnt=true"
	}

	db, _ := models.GetDB()
	defer db.Close()

	var scene models.Scene
	err := db.Preload("Cast").
		Preload("Tags").
		Preload("Cuepoints").
		Preload("Files").
		Where("id = ?", sceneID).First(&scene).Error
	if err != nil {
		log.Error(err)
		return
	}

	var videoFiles []models.File
	videoFiles, err = scene.GetVideoFiles()
	if err != nil {
		log.Error(err)
		return
	}

	if len(videoFiles) == 0 {
		ProcessHeresphereUpdates(&scene, requestData, models.File{})
	} else {
		ProcessHeresphereUpdates(&scene, requestData, videoFiles[0])
	}

	features := make(map[string]bool, 30)
	addFeatureTag := func(feature string) {
		if !features[feature] {
			features[feature] = true
		}
	}

	var media []HeresphereMedia

	videoLength := float64(scene.Duration)

	for i, file := range videoFiles {
		var height = file.VideoHeight
		var width = file.VideoWidth
		var resolution = file.VideoHeight
		var vertresolution = file.VideoWidth

		if file.VideoProjection == "360_tb" {
			resolution = resolution / 2
			vertresolution = vertresolution * 2
		}

		resolutionClass := fmt.Sprintf("%0.fK", math.Round(float64(vertresolution)/1000))
		addFeatureTag("Resolution: " + resolutionClass)

		if file.VideoAvgFrameRateVal > 1.0 {
			addFeatureTag(fmt.Sprintf("Frame Rate: %.0ffps", file.VideoAvgFrameRateVal))
		}

		var mediafile = HeresphereMedia{
			Name: fmt.Sprintf("File %v/%v %vp - %v", i+1, len(videoFiles), resolution, humanize.Bytes(uint64(file.Size))),
			Sources: []HeresphereSource{
				{
					Resolution: resolution,
					Height:     height,
					Width:      width,
					Size:       file.Size,
					URL:        fmt.Sprintf("http://%v/api/dms/file/%v/%v/%v", req.Request.Host, file.ID, scene.GetFunscriptTitle(), dnt),
				},
			},
		}

		media = append(media, mediafile)
		videoLength = file.VideoDuration
	}

	if len(videoFiles) == 0 && config.Config.Web.SceneTrailerlist {
		switch scene.TrailerType {
		case "heresphere":
			heresphereScene := LoadHeresphereScene(scene.TrailerSource)
			media = append(media, heresphereScene.Media...)
		case "slr", "deovr":
			deovrScene := LoadDeovrScene(scene.TrailerSource)
			var hsp HeresphereMedia
			for _, encoding := range deovrScene.Encodings {
				if len(encoding.VideoSources) > 0 {
					hsp.Name = encoding.Name
					for _, source := range encoding.VideoSources {
						hspSource := HeresphereSource{
							URL:        source.URL,
							Width:      source.Width,
							Height:     source.Height,
							Resolution: source.Resolution,
							Size:       source.Size}
						hsp.Sources = append(hsp.Sources, hspSource)
					}
					media = append(media, hsp)
				}
			}

		case "url":
			sources := LoadUrl(scene.TrailerSource)
			media = copyVideoSourceResponse(sources, media)
		case "scrape_html":
			sources := ScrapeHtml(scene.TrailerSource)
			media = copyVideoSourceResponse(sources, media)
		case "scrape_json":
			sources := ScrapeJson(scene.TrailerSource)
			media = copyVideoSourceResponse(sources, media)
		case "load_json":
			sources := LoadJson(scene.TrailerSource)
			media = copyVideoSourceResponse(sources, media)
		}
	}

	if len(videoFiles) > 1 {
		addFeatureTag("Multiple video files")
	}

	var tags []HeresphereTag

	cuepoints := scene.Cuepoints
	sort.Slice(cuepoints, func(i, j int) bool {
		return cuepoints[i].TimeStart < cuepoints[j].TimeStart
	})

	end := 0
	var trackAssignments []string
	for i := range cuepoints {
		start := int(cuepoints[i].TimeStart * 1000)
		if i+1 < len(cuepoints) {
			end = int(cuepoints[i+1].TimeStart*1000) - 1
		} else if int(videoLength*1000) > start {
			end = int(videoLength * 1000)
		} else {
			end = start + 1000
		}

		// split the name into position, action and and extras
		var cuepointNames []string
		if config.Config.Interfaces.Heresphere.MultitrackCuepoints {
			cuepointNames = strings.Split(scene.Cuepoints[i].Name, "-")
		} else {
			cuepointNames = append(cuepointNames, scene.Cuepoints[i].Name)
		}
		for idx, cuepointName := range cuepointNames {
			track := findTrack(idx, len(cuepointNames), cuepointName, &trackAssignments, scene)

			tags = append(tags, HeresphereTag{
				Name:              cuepointName,
				StartMilliseconds: float64(start),
				EndMilliseconds:   float64(end),
				Track:             &track,
			})
		}
	}

	if len(cuepoints) > 1 {
		addFeatureTag("Has cuepoints")
	}

	tags = append(tags, HeresphereTag{
		Name: "Studio:" + scene.Site,
	})

	akaCnt := 0
	for i := range scene.Cast {
		if strings.HasPrefix(scene.Cast[i].Name, "aka:") {
			akaCnt++
			tags = append(tags, HeresphereTag{
				Name: strings.Replace(scene.Cast[i].Name, ",", "/", -1),
			})
		} else {
			tags = append(tags, HeresphereTag{
				Name: "Talent:" + scene.Cast[i].Name,
			})
		}
	}
	if (len(scene.Cast) - akaCnt) > 5 {
		addFeatureTag("Cast: 6+")
	} else if (len(scene.Cast) - akaCnt) > 0 {
		addFeatureTag(fmt.Sprintf("Cast: %d", (len(scene.Cast) - akaCnt)))
	}

	for i := range scene.Tags {
		tags = append(tags, HeresphereTag{
			Name: "Category:" + scene.Tags[i].Name,
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
		addFeatureTag("Is scripted")
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
		addFeatureTag("Has HSP file")
		hspUrl = fmt.Sprintf("http://%v/api/dms/file/%v", req.Request.Host, hspFiles[0].ID)
	}

	var projection string = "equirectangular"
	var stereo string = "sbs"
	var fov = 180.0
	var lens = "Linear"

	if len(videoFiles) == 0 {
		videoFiles = append(videoFiles, models.File{})
	}
	switch videoFiles[0].VideoProjection {
	case "flat":
		addFeatureTag("Flat video")
		projection = "perspective"
		stereo = "mono"

	case "180_mono":
		addFeatureTag("FOV: 180°")
		projection = "equirectangular"
		stereo = "mono"

	case "360_mono":
		addFeatureTag("FOV: 360°")
		projection = "equirectangular360"
		stereo = "mono"

	case "180_sbs":
		addFeatureTag("FOV: 180°")
		projection = "equirectangular"

	case "360_tb":
		addFeatureTag("FOV: 360°")
		projection = "equirectangular360"
		stereo = "tb"

	case "mkx200":
		addFeatureTag("FOV: 200°")
		projection = "fisheye"
		fov = 200.0
		lens = "MKX200"

	case "mkx220":
		addFeatureTag("FOV: 220°")
		projection = "fisheye"
		fov = 220.0
		lens = "MKX220"

	case "vrca220":
		addFeatureTag("FOV: 220°")
		projection = "fisheye"
		fov = 220.0
		lens = "VRCA220"

	case "rf52":
		addFeatureTag("FOV: 190°")
		projection = "fisheye"
		fov = 190.0

	case "fisheye190":
		addFeatureTag("FOV: 190°")
		projection = "fisheye"
		fov = 190.0

	case "fisheye":
		addFeatureTag("FOV: 180°")
		projection = "fisheye"
	}

	title := scene.Title
	thumbnailURL := "http://" + req.Request.Host + "/img/700x/" + strings.Replace(scene.CoverURL, "://", ":/", -1)

	if scene.IsScripted {
		title = scene.GetFunscriptTitle()
		if config.Config.Interfaces.DeoVR.RenderHeatmaps {
			thumbnailURL = "http://" + req.Request.Host + "/imghm/" + fmt.Sprint(scene.ID) + "/" + strings.Replace(scene.CoverURL, "://", ":/", -1)
		}
	}

	if scene.Watchlist {
		addFeatureTag("Watchlist")
	}
	if scene.Trailerlist {
		addFeatureTag("Trailer List")
	}

	if scene.ReleaseDate.Year() > 1900 {
		addFeatureTag("Month: " + scene.ReleaseDate.Format("2006-01"))
		addFeatureTag("Year: " + scene.ReleaseDate.Format("2006"))
	}

	for f, _ := range features {
		tags = append(tags, HeresphereTag{
			Name: "Feature:" + f,
		})
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
		WriteFavorite:        config.Config.Interfaces.Heresphere.AllowFavoriteUpdates,
		WriteRating:          config.Config.Interfaces.Heresphere.AllowRatingUpdates,
		WriteTags:            config.Config.Interfaces.Heresphere.AllowTagUpdates || config.Config.Interfaces.Heresphere.AllowCuepointUpdates || config.Config.Interfaces.Heresphere.AllowWatchlistUpdates || config.Config.Web.SceneTrailerlist,
		WriteHSP:             config.Config.Interfaces.Heresphere.AllowHspData,
	}

	if scene.HasVideoPreview {
		video.ThumbnailVideo = fmt.Sprintf("http://%v/api/dms/preview/%v", req.Request.Host, scene.SceneID)
	}

	resp.WriteHeaderAndEntity(http.StatusOK, video)
}

func copyVideoSourceResponse(sources VideoSourceResponse, media []HeresphereMedia) []HeresphereMedia {
	if len(sources.VideoSources) > 0 {
		for _, source := range sources.VideoSources {
			var hsp HeresphereMedia
			hsp.Name = source.Quality
			hspSource := HeresphereSource{
				URL: source.URL,
			}
			hsp.Sources = append(hsp.Sources, hspSource)
			media = append(media, hsp)
		}
	}
	return media
}

var lockHeresphereUpdates sync.Mutex

func ProcessHeresphereUpdates(scene *models.Scene, requestData HereSphereAuthRequest, videoFile models.File) {

	db, _ := models.GetDB()
	defer db.Close()

	if requestData.IsFavorite != nil && *requestData.IsFavorite != scene.Favourite && config.Config.Interfaces.Heresphere.AllowFavoriteUpdates {
		scene.Favourite = *requestData.IsFavorite
		scene.Save()
	}
	if requestData.Rating != nil && *requestData.Rating != scene.StarRating && config.Config.Interfaces.Heresphere.AllowRatingUpdates {
		scene.StarRating = *requestData.Rating
		scene.Save()
	}

	if requestData.Tags != nil && (config.Config.Interfaces.Heresphere.AllowTagUpdates || config.Config.Interfaces.Heresphere.AllowCuepointUpdates || config.Config.Interfaces.Heresphere.AllowWatchlistUpdates || config.Config.Web.SceneTrailerlist) {
		// need lock, heresphere can send a second post too soon
		lockHeresphereUpdates.Lock()
		defer lockHeresphereUpdates.Unlock()
	}
	if requestData.Tags != nil && config.Config.Interfaces.Heresphere.AllowTagUpdates {
		var newTags []string

		// need to reread the tags, to handle muti threading issues and the scene record may have changed
		// just preload the tags, preload all associations and the scene, does not reread the tags?, so just get them and update the scene
		var tmp models.Scene
		db.Preload("Tags").Where("id = ?", scene.ID).First(&tmp)
		scene.Tags = tmp.Tags

		for _, tag := range *requestData.Tags {
			if strings.HasPrefix(strings.ToLower(tag.Name), "category:") {
				newTags = append(newTags, tag.Name[9:])
			}
		}
		ProcessTagChanges(scene, &newTags, db)
		scene.Save()
	}

	if requestData.Tags != nil && (config.Config.Interfaces.Heresphere.AllowWatchlistUpdates || config.Config.Web.SceneTrailerlist) {
		// need to reread the tags, to handle muti threading issues and the scene record may have changed
		// just preload the tags, preload all associations and the scene, does not reread the tags?, so just get them and update the scene
		var tmp models.Scene
		db.Preload("Tags").Where("id = ?", scene.ID).First(&tmp)
		scene.Tags = tmp.Tags

		watchlist := false
		trailerlist := false
		for _, tag := range *requestData.Tags {
			if strings.HasPrefix(strings.ToLower(tag.Name), "feature:watchlist") {
				watchlist = true
			}
			if strings.HasPrefix(strings.ToLower(tag.Name), "feature:trailer list") {
				trailerlist = true
			}
		}
		if scene.Watchlist != watchlist && config.Config.Interfaces.Heresphere.AllowWatchlistUpdates {
			scene.Watchlist = watchlist
			scene.Save()
		}
		if scene.Trailerlist != trailerlist && config.Config.Web.SceneTrailerlist {
			scene.Trailerlist = trailerlist
			scene.Save()
		}
	}

	if requestData.Tags != nil && config.Config.Interfaces.Heresphere.AllowCuepointUpdates {
		// need to reread the cuepoints, to handle muti threading issues and the scene record may have changed
		// just preload the cuepoint, preload all associations and the scene, does not reread the cuepoint?, so just get them and update the scene
		var existingScene models.Scene
		db.Preload("Cuepoints").Where("id = ?", scene.ID).First(&existingScene)

		var replacementCuepoints []models.SceneCuepoint
		endpos := findEndPos(requestData)
		firstTrack := findTheMainTrack(requestData)
		// build new list of cuepoints
		for _, tag := range *requestData.Tags {
			if !strings.Contains(tag.Name, ":") {
				if *tag.Track == firstTrack {
					replacementCuepoints = append(replacementCuepoints, models.SceneCuepoint{SceneID: scene.ID, TimeStart: float64(tag.StartMilliseconds) / 1000, Name: tag.Name})
				} else {
					//allow for multi track, merge into the main cuepoint name
					if tag.StartMilliseconds > 0 || tag.EndMilliseconds < endpos {
						for idx, newtag := range replacementCuepoints {
							// allow 5 seconds lewway to align manually entered tags
							if math.Abs((newtag.TimeStart)-tag.StartMilliseconds/1000) < 5 {
								replacementCuepoints[idx].Name = tag.Name + "-" + replacementCuepoints[idx].Name
							}
						}
					}
				}
			}
		}

		// workout cuepoints differences and append/delete
		// note association.replace does not work, it just changes cuepoint records and sets the sceneid to null on the existing cuepoint record, l;eaving them in the db
		for _, newCuepoint := range replacementCuepoints {
			if matchCuepoint(newCuepoint, existingScene.Cuepoints) == -1 {
				db.Model(&scene).Association("Cuepoints").Append(&newCuepoint)
			}
		}
		for _, existingCuepoint := range existingScene.Cuepoints {
			if matchCuepoint(existingCuepoint, replacementCuepoints) == -1 {
				db.Model(&existingCuepoint).Delete(&existingCuepoint)
			}
		}
	}

	if requestData.DeleteFiles != nil && config.Config.Interfaces.Heresphere.AllowFileDeletes {
		for _, sceneFile := range scene.Files {
			removeFileByFileId(sceneFile.ID)
		}
	}

	if requestData.Hsp != nil && config.Config.Interfaces.Heresphere.AllowHspData {
		hspContent, err := base64.StdEncoding.DecodeString(*requestData.Hsp)
		if err != nil {
			log.Error("Error decoding heresphere hsp data %v", err)
		}

		fName := filepath.Join(scene.Files[0].Path, strings.TrimSuffix(scene.Files[0].Filename, filepath.Ext(videoFile.Filename))+".hsp")
		ioutil.WriteFile(fName, hspContent, 0644)

		tasks.ScanLocalHspFile(fName, videoFile.VolumeID, scene.ID)
	}
}
func findTheMainTrack(requestData HereSphereAuthRequest) int {
	// 99% of the time we want Track 0, but the user may have deleted and added whole track

	// find the max duration
	endpos := findEndPos(requestData)
	for _, tag := range *requestData.Tags {
		if endpos < tag.EndMilliseconds {
			endpos = tag.EndMilliseconds
		}
	}

	// find the best track
	likelyTrack := 9999
	alternateTrack := 9999

	for _, tag := range *requestData.Tags {
		if (tag.StartMilliseconds > 0 || tag.EndMilliseconds < endpos) && !strings.Contains(tag.Name, ":") {
			return *tag.Track
		}

		if (tag.StartMilliseconds > 0 || tag.EndMilliseconds < endpos) && likelyTrack > *tag.Track {
			likelyTrack = *tag.Track
		}
		if !strings.Contains(tag.Name, ":") && alternateTrack > *tag.Track {
			alternateTrack = *tag.Track
		}
	}

	if likelyTrack < 9999 {
		return likelyTrack
	}

	if alternateTrack < 9999 {
		return likelyTrack
	}

	return -1
}
func findEndPos(requestData HereSphereAuthRequest) float64 {
	// find the max duration
	endpos := float64(0)
	for _, tag := range *requestData.Tags {
		if endpos < tag.EndMilliseconds {
			endpos = tag.EndMilliseconds
		}
	}
	return endpos
}
func matchCuepoint(findCuepoint models.SceneCuepoint, cuepointList []models.SceneCuepoint) int {
	for idx, cuepoint := range cuepointList {
		if cuepoint.Name == findCuepoint.Name && cuepoint.TimeStart == findCuepoint.TimeStart {
			return idx
		}
	}
	return -1
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
			r.Limit = optional.NewInt(20000)

			q := models.QueryScenes(r, false)

			list := make([]string, len(q.Scenes))
			for i := range q.Scenes {
				url := fmt.Sprintf("http://%v/heresphere/%v", req.Request.Host, q.Scenes[i].ID)
				list[i] = url
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

	if len(unmatched) > 0 {
		list := make([]string, len(unmatched))
		for i := range unmatched {
			url := fmt.Sprintf("http://%v/heresphere/file/%v", req.Request.Host, unmatched[i].ID)
			list[i] = url
		}

		sceneLists = append(sceneLists, HeresphereListScenes{
			Name: "Unmatched",
			List: list,
		})

	}

	if config.Config.Web.SceneTrailerlist {
		var trailerlist []models.Scene
		db.Preload("Tags").Where("is_available = false and trailer_source != '' and (trailerlist )").Order("updated_at desc").Find(&trailerlist)
		if len(trailerlist) > 0 {
			var list []string
			for i := range trailerlist {
				downloadTag := false
				for _, tag := range trailerlist[i].Tags {
					if tag.Name == "download" {
						downloadTag = true
					}
				}
				if !downloadTag {
					url := fmt.Sprintf("http://%v/heresphere/%v", req.Request.Host, trailerlist[i].ID)
					list = append(list, url)
				}
			}

			sceneLists = append(sceneLists, HeresphereListScenes{
				Name: "Trailers",
				List: list,
			})

		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, HeresphereLibrary{
		Access:  1,
		Library: sceneLists,
	})
}

func findTrack(indexpos int, cuepointCount int, name string, trackAssignments *[]string, scene models.Scene) int {
	// find the track number to use for the actor, actors get their own track
	if isCast(name, scene) {
		return getTrackAssignment(name, trackAssignments)
	}

	// if a position & action exist, make the action track zero, it is more likely the main cuepoint
	if cuepointCount > 1 {
		if indexpos == 0 {
			return getTrackAssignment(fmt.Sprintf("group%v", indexpos), trackAssignments)
		}
		if indexpos == 1 {
			return 0
		}
		getTrackAssignment(fmt.Sprintf("group%v", indexpos), trackAssignments)
	}
	return 0
}

func isCast(name string, scene models.Scene) bool {
	// check if the cuepoint is the same as a cast members name
	for _, cast := range scene.Cast {
		if strings.EqualFold(cast.Name, name) {
			return true
		}
	}
	return false
}

func getTrackAssignment(name string, trackAssignments *[]string) int {
	// find tracks allocated track or add a new one
	for idx, track := range *trackAssignments {
		if strings.EqualFold(track, name) {
			return idx + 1
		}
	}
	*trackAssignments = append(*trackAssignments, name)
	return len(*trackAssignments)
}
