package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/avast/retry-go/v4"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"

	"github.com/xbapps/xbvr/pkg/common"
)

// SceneCuepoint data model
type SceneCuepoint struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	SceneID   uint    `gorm:"index" json:"-" xbvrbackup:"-"`
	TimeStart float64 `json:"time_start" xbvrbackup:"time_start"`
	TimeEnd   float64 `json:"time_end,omitempty" xbvrbackup:"time_end"`
	Track     *uint   `json:"track,omitempty" xbvrbackup:"track"`
	Name      string  `json:"name" xbvrbackup:"name"`
	IsHSP     string  `gorm:"-" json:"is_hsp" xbvrbackup:"-"`
	Rating    float64 `json:"rating" xbvrbackup:"rating"`
}

func (o *SceneCuepoint) Save() error {
	commonDb, _ := GetCommonDB()

	var err error = retry.Do(
		func() error {
			err := commonDb.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

// Scene data model
type Scene struct {
	ID        uint       `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time  `json:"created_at" xbvrbackup:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" xbvrbackup:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"-" xbvrbackup:"-"`

	SceneID         string    `gorm:"index" json:"scene_id" xbvrbackup:"scene_id"`
	Title           string    `json:"title" sql:"type:varchar(1024);" xbvrbackup:"title"`
	SceneType       string    `json:"scene_type" xbvrbackup:"scene_type"`
	ScraperId       string    `json:"scraper_id" xbvrbackup:"scraper_id"`
	Studio          string    `json:"studio" xbvrbackup:"studio"`
	Site            string    `json:"site" xbvrbackup:"site"`
	Tags            []Tag     `gorm:"many2many:scene_tags;" json:"tags" xbvrbackup:"tags"`
	Cast            []Actor   `gorm:"many2many:scene_cast;" json:"cast" xbvrbackup:"cast"`
	FilenamesArr    string    `json:"filenames_arr" sql:"type:text;" xbvrbackup:"filenames_arr"`
	Images          string    `json:"images" sql:"type:text;" xbvrbackup:"images"`
	Files           []File    `json:"file" xbvrbackup:"-"`
	Duration        int       `json:"duration" xbvrbackup:"duration"`
	Synopsis        string    `json:"synopsis" sql:"type:text;" xbvrbackup:"synopsis"`
	ReleaseDate     time.Time `json:"release_date" xbvrbackup:"release_date"`
	ReleaseDateText string    `json:"release_date_text" xbvrbackup:"release_date_text"`
	CoverURL        string    `gorm:"size:500" json:"cover_url" xbvrbackup:"cover_url"`
	SceneURL        string    `gorm:"size:500" json:"scene_url" xbvrbackup:"scene_url"`
	MemberURL       string    `json:"members_url" xbvrbackup:"members_url"`
	IsMultipart     bool      `json:"is_multipart" xbvrbackup:"is_multipart"`

	StarRating     float64         `json:"star_rating" xbvrbackup:"star_rating"`
	Favourite      bool            `json:"favourite" gorm:"default:false" xbvrbackup:"favourite"`
	Watchlist      bool            `json:"watchlist" gorm:"default:false" xbvrbackup:"watchlist"`
	Wishlist       bool            `json:"wishlist" gorm:"default:false" xbvrbackup:"wishlist"`
	IsAvailable    bool            `json:"is_available" gorm:"default:false" xbvrbackup:"-"`
	IsAccessible   bool            `json:"is_accessible" gorm:"default:false" xbvrbackup:"-"`
	IsWatched      bool            `json:"is_watched" gorm:"default:false" xbvrbackup:"is_watched"`
	IsScripted     bool            `json:"is_scripted" gorm:"default:false" xbvrbackup:"-"`
	Cuepoints      []SceneCuepoint `json:"cuepoints" xbvrbackup:"-"`
	History        []History       `json:"history" xbvrbackup:"-"`
	AddedDate      time.Time       `json:"added_date" xbvrbackup:"added_date"`
	LastOpened     time.Time       `json:"last_opened" xbvrbackup:"last_opened"`
	TotalFileSize  int64           `json:"total_file_size" xbvrbackup:"-"`
	TotalWatchTime int             `json:"total_watch_time" gorm:"default:0" xbvrbackup:"total_watch_time"`

	HasVideoPreview bool `json:"has_preview" gorm:"default:false" xbvrbackup:"-"`
	// HasVideoThumbnail bool `json:"has_video_thumbnail" gorm:"default:false"`

	NeedsUpdate   bool   `json:"needs_update" xbvrbackup:"-"`
	EditsApplied  bool   `json:"edits_applied" gorm:"default:false" xbvrbackup:"-"`
	TrailerType   string `json:"trailer_type" xbvrbackup:"trailer_type"`
	TrailerSource string `json:"trailer_source" sql:"type:longtext;"  xbvrbackup:"trailer_source"`
	ChromaKey     string `json:"passthrough" xbvrbackup:"passthrough"`
	Trailerlist   bool   `json:"trailerlist" gorm:"default:false" xbvrbackup:"trailerlist"`
	IsSubscribed  bool   `json:"is_subscribed" gorm:"default:false"`
	IsHidden      bool   `json:"is_hidden" gorm:"default:false" xbvrbackup:"is_hidden"`
	LegacySceneID string `json:"legacy_scene_id" xbvrbackup:"legacy_scene_id"`

	ScriptPublished time.Time `json:"script_published" xbvrbackup:"script_published"`
	AiScript        bool      `json:"ai_script" gorm:"default:false" xbvrbackup:"ai_script"`
	HumanScript     bool      `json:"human_script" gorm:"default:false" xbvrbackup:"human_script"`

	Description string  `gorm:"-" json:"description" xbvrbackup:"-"`
	Score       float64 `gorm:"-" json:"_score" xbvrbackup:"-"`

	AlternateSource []ExternalReferenceLink `json:"alternate_source" xbvrbackup:"-"`
}

type Image struct {
	URL         string `json:"url"`
	Type        string `json:"type"`
	Orientation string `json:"orientation"`
}

type VideoSourceResponse struct {
	VideoSources []VideoSource `json:"video_sources"`
}

type VideoSource struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

type Config struct {
	Advanced struct {
		UseAltSrcInFileMatching bool `json:"useAltSrcInFileMatching"`
	} `json:"advanced"`
}

func (i *Scene) Save() error {
	commonDb, _ := GetCommonDB()

	var err error = retry.Do(
		func() error {
			err := commonDb.Save(&i).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

func (i *Scene) ToJSON() ([]byte, error) {
	return json.Marshal(i)
}

func (i *Scene) FromJSON(data []byte) error {
	return json.Unmarshal(data, &i)
}

func (o *Scene) GetIfExist(id string) error {
	commonDb, _ := GetCommonDB()

	return commonDb.
		Preload("Tags").
		Preload("Cast").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints").
		Where(&Scene{SceneID: id}).First(o).Error
}

func (o *Scene) GetIfExistByPK(id uint) error {
	commonDb, _ := GetCommonDB()

	return commonDb.
		Preload("Tags").
		Preload("Cast").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints").
		Where(&Scene{ID: id}).First(o).Error
}

func (o *Scene) GetIfExistURL(u string) error {
	commonDb, _ := GetCommonDB()

	return commonDb.
		Preload("Tags").
		Preload("Cast").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints").
		Where(&Scene{SceneURL: u}).First(o).Error
}

func (o *Scene) GetFunscriptTitle() string {
	// first make the title filename safe
	re := regexp.MustCompile(`[?/\<>|]`)

	title := o.Title
	// Colons are pretty common in titles, so we use a unicode alternative
	title = strings.ReplaceAll(title, ":", "꞉")
	// all other unsafe characters get removed
	title = re.ReplaceAllString(title, "")

	// add ID to prevent title collisions
	return fmt.Sprintf("%d - %s", o.ID, title)
}

func (o *Scene) GetFiles() ([]File, error) {
	commonDb, _ := GetCommonDB()

	var files []File
	commonDb.Preload("Volume").Where(&File{SceneID: o.ID}).Find(&files)

	return files, nil
}

func (o *Scene) GetTotalWatchTime() int {
	commonDb, _ := GetCommonDB()

	totalResult := struct{ Total float64 }{}
	commonDb.Raw(`select sum(duration) as total from histories where scene_id = ?`, o.ID).Scan(&totalResult)

	return int(totalResult.Total)
}

func (o *Scene) GetVideoFiles() ([]File, error) {
	files, err := o.GetVideoFilesSorted("")
	return files, err
}

func (o *Scene) GetVideoFilesSorted(sort string) ([]File, error) {
	commonDb, _ := GetCommonDB()

	var files []File
	if sort == "" {
		commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "video").Find(&files)
	} else {
		commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "video").Order(sort).Find(&files)
	}

	return files, nil
}

func (o *Scene) GetScriptFiles() ([]File, error) {
	files, err := o.GetScriptFilesSorted("is_selected_script DESC, created_time DESC")
	return files, err
}

func (o *Scene) GetScriptFilesSorted(sort string) ([]File, error) {
	commonDb, _ := GetCommonDB()

	var files []File
	if sort == "" {
		commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "script").Find(&files)
	} else {
		commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "script").Order(sort).Find(&files)
	}

	return files, nil
}

func (o *Scene) GetHSPFiles() ([]File, error) {
	commonDb, _ := GetCommonDB()

	var files []File
	commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "hsp").Find(&files)

	return files, nil
}

func (o *Scene) GetSubtitlesFilesSorted(sort string) ([]File, error) {
	commonDb, _ := GetCommonDB()

	var files []File
	if sort == "" {
		commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "subtitles").Find(&files)
	} else {
		commonDb.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "subtitles").Order(sort).Find(&files)
	}

	return files, nil
}

func (o *Scene) PreviewExists() bool {
	if _, err := os.Stat(filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", o.SceneID))); os.IsNotExist(err) {
		return false
	}
	return true
}

func (o *Scene) UpdateStatus() {
	// Check if file with scene association exists
	files, err := o.GetFiles()
	if err != nil {
		return
	}

	changed := false
	scripts := 0
	videos := 0

	if len(files) > 0 {
		var newestFileDate time.Time
		var totalFileSize int64
		anyVideoAccessible := false

		for j := range files {
			totalFileSize = totalFileSize + files[j].Size

			if files[j].Type == "script" {
				scripts = scripts + 1

				if files[j].Exists() && (files[j].CreatedTime.After(newestFileDate) || newestFileDate.IsZero()) {
					newestFileDate = files[j].CreatedTime
				}
			}

			if files[j].Type == "video" {
				videos = videos + 1

				if files[j].Exists() {
					anyVideoAccessible = true

					if files[j].CreatedTime.After(newestFileDate) || newestFileDate.IsZero() {
						newestFileDate = files[j].CreatedTime
					}
				}
			}
		}

		if totalFileSize != o.TotalFileSize {
			o.TotalFileSize = totalFileSize
			changed = true
		}

		if scripts > 0 && !o.IsScripted {
			o.IsScripted = true
			changed = true
		}

		if scripts == 0 && o.IsScripted {
			o.IsScripted = false
			changed = true
		}

		if anyVideoAccessible != o.IsAccessible {
			o.IsAccessible = anyVideoAccessible
			changed = true
		}

		if videos > 0 && !o.IsAvailable {
			o.IsAvailable = true
			o.Wishlist = false
			changed = true
		}

		if videos == 0 && o.IsAvailable {
			o.IsAvailable = false
			changed = true
		}

		if !newestFileDate.Equal(o.AddedDate) && !newestFileDate.IsZero() {
			o.AddedDate = newestFileDate
			changed = true
		}
	} else {
		if o.IsAvailable {
			o.IsAvailable = false
			changed = true
		}

		if o.IsScripted {
			o.IsScripted = false
			changed = true
		}
	}

	if o.HasVideoPreview && !o.PreviewExists() {
		o.HasVideoPreview = false
		changed = true
	}

	if !o.HasVideoPreview && o.PreviewExists() {
		o.HasVideoPreview = true
		changed = true
	}

	totalWatchTime := o.GetTotalWatchTime()
	if o.TotalWatchTime != totalWatchTime {
		o.TotalWatchTime = totalWatchTime
		changed = true
	}

	if changed {
		o.Save()
	}
}

func SceneCreateUpdateFromExternal(db *gorm.DB, ext ScrapedScene) error {
	if ext.SceneID == "" {
		return nil
	}

	var o Scene
	db.Where(&Scene{SceneID: ext.SceneID}).FirstOrCreate(&o)

	if o.Title != ext.Title {
		// reset scriptfile.IsExported state on title change
		scriptfiles, err := o.GetScriptFiles()
		if err == nil {
			for _, file := range scriptfiles {
				if file.IsExported {
					file.IsExported = false
					file.Save()
				}
			}
		}
	}

	o.PopulateSceneFieldsFromExternal(db, ext)
	var site Site
	db.Where("id = ?", o.ScraperId).FirstOrInit(&site)
	o.IsSubscribed = site.Subscribed

	// Clean & Associate Tags
	var tags = o.Tags
	db.Model(&o).Association("Tags").Clear()
	for idx, tag := range tags {
		tmpTag := Tag{}
		db.Where(&Tag{Name: tag.Name}).FirstOrCreate(&tmpTag)
		tags[idx] = tmpTag
	}
	o.Tags = tags
	SaveWithRetry(db, &o)

	// Clean & Associate Actors
	db.Model(&o).Association("Cast").Clear()
	var tmpActor Actor
	for _, name := range ext.Cast {
		tmpActor = Actor{}
		db.Where(&Actor{Name: strings.Replace(name, ".", "", -1)}).FirstOrCreate(&tmpActor)
		saveActor := false
		if ext.ActorDetails[name].ImageUrl != "" {
			if tmpActor.ImageUrl == "" {
				tmpActor.ImageUrl = ext.ActorDetails[name].ImageUrl
				saveActor = true
			}
			if tmpActor.AddToImageArray(ext.ActorDetails[name].ImageUrl) {
				saveActor = true
			}
		}
		if ext.ActorDetails[name].ProfileUrl != "" {
			if strings.HasPrefix(ext.ActorDetails[name].ProfileUrl, "https://stashdb.org/performers/") {

			} else {
				if tmpActor.AddToActorUrlArray(ActorLink{Url: ext.ActorDetails[name].ProfileUrl, Type: ext.ActorDetails[name].Source}) {
					saveActor = true
				}
			}
		}
		if saveActor {
			tmpActor.Save()
		}
		db.Model(&o).Association("Cast").Append(tmpActor)
	}
	// delete any altrernate scene records, in case this scene was originally a linked scene
	var extrefs []ExternalReference
	db.Where("external_source like 'alternate scene %' and external_url = ?", o.SceneURL).Find(&extrefs)
	for _, extref := range extrefs {
		db.Where("external_reference_id = ?", extref.ID).Delete(&ExternalReferenceLink{})
		db.Delete(&extref)
	}

	return nil
}

func (o *Scene) PopulateSceneFieldsFromExternal(db *gorm.DB, ext ScrapedScene) {
	// this function is shared between scenes and alternate scenes,
	//	it should only setup values in the scene record from the scraped scene
	//	it should not update scene data, as that won't apply for alternate scene sources
	if ext.SceneID == "" {
		return
	}

	o.NeedsUpdate = false
	o.EditsApplied = false
	o.SceneID = ext.SceneID
	o.ScraperId = ext.ScraperID
	o.Title = ext.Title
	o.SceneType = ext.SceneType
	o.Studio = ext.Studio
	o.Site = ext.Site
	o.Duration = ext.Duration
	o.Synopsis = ext.Synopsis
	o.ReleaseDateText = ext.Released
	if ext.Covers != nil {
		o.CoverURL = ext.Covers[0]
	}
	o.SceneURL = ext.HomepageURL
	o.MemberURL = ext.MembersUrl

	o.ChromaKey = ext.ChromaKey
	if ext.HasScriptDownload && o.ScriptPublished.IsZero() {
		o.ScriptPublished = time.Now()
	}
	o.AiScript = ext.AiScript
	o.HumanScript = ext.HumanScript

	// Trailers
	o.TrailerType = ext.TrailerType
	o.TrailerSource = ext.TrailerSrc

	if ext.Released != "" {
		dateParsed, err := dateparse.ParseLocal(strings.Replace(ext.Released, ",", "", -1))
		if err == nil {
			o.ReleaseDate = dateParsed
		}
	}

	// Store filenames as JSON
	pfTxt, err := json.Marshal(ext.Filenames)
	if err == nil {
		o.FilenamesArr = string(pfTxt)
	}

	// Store images as JSON
	var images []Image

	for i := range ext.Covers {
		if ext.Covers[i] != "" {
			images = append(images, Image{
				URL:  ext.Covers[i],
				Type: "cover",
			})
		}
	}

	for i := range ext.Gallery {
		if ext.Gallery[i] != "" {
			images = append(images, Image{
				URL:  ext.Gallery[i],
				Type: "gallery",
			})
		}
	}

	imgTxt, err := json.Marshal(images)
	if err == nil {
		o.Images = string(imgTxt)
	}

	var site Site
	db.Where("id = ?", o.ScraperId).FirstOrInit(&site)
	o.IsSubscribed = site.Subscribed

	var tags []Tag
	for _, name := range ext.Tags {
		tagClean := ConvertTag(name)
		if tagClean != "" {
			tags = append(tags, Tag{Name: tagClean})
		}
	}
	o.Tags = tags

	// Clean & Associate Actors
	var cast []Actor
	var tmpActor Actor
	for _, name := range ext.Cast {
		tmpActor = Actor{}
		db.Where(&Actor{Name: strings.Replace(name, ".", "", -1)}).FirstOrCreate(&tmpActor)
		cast = append(cast, tmpActor)
	}
	o.Cast = cast
}

func SceneUpdateScriptData(db *gorm.DB, ext ScrapedScene) {
	if ext.MasterSiteId == "" {
		var o Scene
		o.GetIfExistByPK(ext.InternalSceneId)

		if o.ID != 0 {
			if o.ScriptPublished.IsZero() || o.HumanScript != ext.HumanScript || o.AiScript != ext.AiScript {
				o.ScriptPublished = time.Now()
				o.HumanScript = ext.HumanScript
				o.AiScript = ext.AiScript
				o.Save()
			}
		}
	} else {
		var extref ExternalReference
		extref.FindExternalId("alternate scene "+ext.ScraperID, ext.SceneID)
		var externalData SceneAlternateSource
		json.Unmarshal([]byte(extref.ExternalData), &externalData)
		if extref.ID > 0 {
			if externalData.Scene.ScriptPublished.IsZero() || externalData.Scene.HumanScript != ext.HumanScript || externalData.Scene.AiScript != ext.AiScript {
				// set user defined fields for querying, rather than querying the json
				extref.UdfDatetime1 = time.Now()
				extref.UdfBool1 = ext.HumanScript
				extref.UdfBool2 = ext.AiScript
				externalData.Scene.ScriptPublished = extref.UdfDatetime1
				externalData.Scene.HumanScript = ext.HumanScript
				externalData.Scene.AiScript = ext.AiScript
				newjson, _ := json.Marshal(externalData)
				extref.ExternalData = string(newjson)
				extref.Save()
			}
		}
	}
}

type RequestSceneList struct {
	DlState      optional.String   `json:"dlState"`
	Limit        optional.Int      `json:"limit"`
	Offset       optional.Int      `json:"offset"`
	IsAvailable  optional.Bool     `json:"isAvailable"`
	IsAccessible optional.Bool     `json:"isAccessible"`
	IsWatched    optional.Bool     `json:"isWatched"`
	Lists        []optional.String `json:"lists"`
	Sites        []optional.String `json:"sites"`
	Tags         []optional.String `json:"tags"`
	Cast         []optional.String `json:"cast"`
	Cuepoint     []optional.String `json:"cuepoint"`
	Attributes   []optional.String `json:"attributes"`
	Volume       optional.Int      `json:"volume"`
	Released     optional.String   `json:"releaseMonth"`
	Sort         optional.String   `json:"sort"`
}

type ResponseSceneList struct {
	Results            int     `json:"results"`
	Scenes             []Scene `json:"scenes"`
	CountAny           int     `json:"count_any"`
	CountAvailable     int     `json:"count_available"`
	CountDownloaded    int     `json:"count_downloaded"`
	CountNotDownloaded int     `json:"count_not_downloaded"`
	CountHidden        int     `json:"count_hidden"`
}

func QueryScenesFull(r RequestSceneList) ResponseSceneList {
	var scenes []Scene
	r.Limit = optional.NewInt(100)
	r.Offset = optional.NewInt(0)

	q := QueryScenes(r, true)
	scenes = q.Scenes

	for len(scenes) < q.Results {
		r.Offset = optional.NewInt(len(scenes))
		q := QueryScenes(r, true)
		scenes = append(scenes, q.Scenes...)
	}

	q.Scenes = scenes
	return q
}

func QueryScenes(r RequestSceneList, enablePreload bool) ResponseSceneList {
	r.Limit = optional.NewInt(r.Limit.OrElse(100))

	db, _ := GetDB()
	defer db.Close()

	preCountTx, finalTx := queryScenes(db, r)

	var out ResponseSceneList

	// Count other variations
	preCountTx.Where("is_hidden = ?", false).Count(&out.CountAny)
	preCountTx.Where("is_available = ?", true).Where("is_accessible = ?", true).Where("is_hidden = ?", false).Count(&out.CountAvailable)
	preCountTx.Where("is_available = ?", true).Where("is_hidden = ?", false).Count(&out.CountDownloaded)
	preCountTx.Where("is_available = ?", false).Where("is_hidden = ?", false).Count(&out.CountNotDownloaded)
	preCountTx.Where("is_hidden = ?", true).Count(&out.CountHidden)

	// r.Offset must _not_ apply to the count, as the count query always returns a single value
	finalTx.Offset(0).Count(&out.Results)

	if enablePreload {
		finalTx = finalTx.
			Preload("Cast").
			Preload("Tags").
			Preload("Files").
			Preload("History").
			Preload("Cuepoints")
	}
	finalTx.Find(&out.Scenes)

	return out
}

func QuerySceneIDs(r RequestSceneList) []string {
	db, _ := GetDB()
	defer db.Close()

	_, finalTx := queryScenes(db, r)

	var ids []string
	finalTx.Pluck("scenes.id", &ids)

	return ids
}

type SceneSummary struct {
	ID         uint
	Title      string
	Duration   uint
	CoverURL   string
	IsScripted bool
}

func QuerySceneSummaries(r RequestSceneList) []SceneSummary {
	db, _ := GetDB()
	defer db.Close()

	_, finalTx := queryScenes(db, r)

	var summaries []SceneSummary
	finalTx.Select("scenes.id, title, duration, cover_url, is_scripted").Scan(&summaries)

	return summaries
}

func queryScenes(db *gorm.DB, r RequestSceneList) (*gorm.DB, *gorm.DB) {
	// get config, can't reference config directly due to circular package references
	config := getConfig(db)

	tx := db.Model(&Scene{})

	if r.IsWatched.Present() {
		tx = tx.Where("is_watched = ?", r.IsWatched.OrElse(true))
	}

	if r.Volume.Present() && r.Volume.OrElse(0) != 0 {
		tx = tx.
			Joins("left join files on files.scene_id=scenes.id").
			Where("files.volume_id = ?", r.Volume.OrElse(0))
	}

	for _, i := range r.Lists {
		if i.OrElse("") == "watchlist" {
			tx = tx.Where("scenes.watchlist = ?", true)
		}
		if i.OrElse("") == "favourite" {
			tx = tx.Where("scenes.favourite = ?", true)
		}
		if i.OrElse("") == "wishlist" {
			tx = tx.Where("wishlist = ?", true)
		}
		if i.OrElse("") == "scripted" {
			tx = tx.Where("is_scripted = ?", true)
		}
	}

	// handle Attribute selections
	var orAttribute []string
	var andAttribute []string
	combinedWhere := ""
	for _, attribute := range r.Attributes {
		fieldName := attribute.OrElse("")

		negate := strings.HasPrefix(fieldName, "!")   // ! prefix indicate NOT filtering
		mustHave := strings.HasPrefix(fieldName, "&") // & prefix indicate must have filtering
		if negate || mustHave {
			fieldName = fieldName[1:]
		}

		value := ""
		if strings.HasPrefix(fieldName, "Resolution ") {
			value = strings.Replace(fieldName[11:], "K", "", 1)
			fieldName = "Resolution"
		}
		if strings.HasPrefix(fieldName, "Frame Rate ") {
			value = strings.Replace(fieldName[11:], "fps", "", 1)
			fieldName = "Frame Rate"
		}
		if strings.HasPrefix(fieldName, "Codec ") {
			value = fieldName[6:]
			fieldName = "Codec"
		}
		if strings.HasPrefix(fieldName, "Rating ") {
			value = fieldName[7:]
			fieldName = "Rating"
		}

		where := ""
		switch fieldName {
		case "Multiple Video Files":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' group by files.scene_id having count(*) > 1)"
		case "Single Video File":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' group by files.scene_id having count(*) = 1)"
		case "Multiple Script Files":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'script' group by files.scene_id having count(*) > 1)"
		case "Single Script File":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'script' group by files.scene_id having count(*) = 1)"
		case "Has Hsp File":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'hsp')"
		case "Has Subtitles File":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'subtitles')"
		case "Has Rating":
			where = "scenes.star_rating > 0"
		case "Has Cuepoints":
			where = "exists (select 1 from scene_cuepoints where scene_cuepoints.scene_id = scenes.id)"
		case "Has Simple Cuepoints":
			where = "exists (select 1 from scene_cuepoints where scene_cuepoints.scene_id = scenes.id and track is null)"
		case "Has HSP Cuepoints":
			where = "exists (select 1 from scene_cuepoints where scene_cuepoints.scene_id = scenes.id and track is not null)"
		case "In Trailer List":
			where = "trailerlist = 1"
		case "Has Preview":
			where = "has_video_preview = 1"
		case "Has Subscription":
			where = "is_subscribed = 1"
		case "Rating":
			where = "scenes.star_rating = " + value
		case "No Actor/Cast":
			where = "exists (select 1 from scenes s left join scene_cast sc on sc.scene_id =s.id where s.id=scenes.id and  sc.scene_id is NULL)"
		case "Cast 6+":
			where = "exists (select 1 from scene_cast join actors on actors.id = scene_cast.actor_id where scene_cast.scene_id = scenes.id and actors.name not like 'aka:%' group by scene_cast.scene_id having count(*) > 5)"
		case "Cast 1", "Cast 2", "Cast 3", "Cast 4", "Cast 5":
			where = "exists (select 1 from scene_cast join actors on actors.id = scene_cast.actor_id where scene_cast.scene_id = scenes.id and actors.name not like 'aka:%' group by scene_cast.scene_id having count(*) = " + fieldName[5:] + ")"
		case "Resolution":
			div := "/"
			if tx.Dialect().GetName() == "mysql" {
				div = "div"
			}
			where = "exists (select 1 from files where files.scene_id = scenes.id and ((files.video_width * (case when files.video_projection like '%_tb' then 2 else 1 end) + 500) " + div + " 1000) = " + value + ")"
		case "Frame Rate":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_avg_frame_rate_val = " + value + ")"
		case "Flat video":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection = 'flat')"
		case "FOV: 180°":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('180_mono','180_sbs','fisheye'))"
		case "FOV: 190°":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('rf52','fisheye190'))"
		case "FOV: 200°":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection = 'mkx200')"
		case "FOV: 220°":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('mkx220','vrca220'))"
		case "FOV: 360°":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('360_mono','360_tb'))"
		case "Projection Perspective":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection = 'flat')"
		case "Projection Equirectangular":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('180_mono','180_sbs'))"
		case "Projection Equirectangular360":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('360_tb','360_mono'))"
		case "Projection Fisheye":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('mkx200','mkx220','vrca220','rf52','fisheye190','fisheye'))"
		case "Mono":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('flat','180_mono','360_mono'))"
		case "Top/Bottom":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection in ('180_tb','360_tb'))"
		case "Side by Side":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection not in (flat','180_mono','360_mono', '180_tb', '360_tb'))"
		case "MKX200":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection = 'mkx200')"
		case "MKX220":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection = 'mkx220')"
		case "VRCA220":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_projection = 'vrca220')"
		case "Codec":
			where = "exists (select 1 from files where files.scene_id = scenes.id and files.`type` = 'video' and files.video_codec_name = '" + value + "')"
		case "In Watchlist":
			where = "scenes.watchlist = 1"
		case "Is Scripted":
			where = "is_scripted = 1"
		case "Is Favourite":
			where = "scenes.favourite = 1"
		case "Is Passthrough":
			where = "chroma_key <> ''"
		case "Is Alpha Passthrough":
			where = `chroma_key <> '' and chroma_key like '%"hasAlpha":true%'`
		case "In Wishlist":
			where = "wishlist = 1"
		case "Stashdb Linked":
			where = "exists (select 1 from external_reference_links erl where erl.internal_db_id = scenes.id and erl.external_source = 'stashdb scene')"
		case "POVR Scraper":
			where = `scenes.scene_id like "povr-%"`
		case "SLR Scraper":
			where = `scenes.scene_id like "slr-%"`
		case "Has Image":
			where = "cover_url not in ('','http://localhost/dont_cause_errors')"
		case "VRPHub Scraper":
			where = `scenes.scene_id like "vrphub-%"`
		case "VRPorn Scraper":
			where = `scenes.scene_id like "vrporn-%"`
		case "Has Script Download":
			// querying the scenes in from alternate sources (stored in external_reference) has a performance impact, so it's user choice
			if config.Advanced.UseAltSrcInFileMatching {
				where = "(scenes.script_published > '0001-01-01 00:00:00+00:00' or (select distinct 1 from external_reference_links erl join external_references er on er.id=erl.external_reference_id where erl.internal_table='scenes' and internal_db_id=scenes.id and er.udf_datetime1 > '0001-01-02'))"
			} else {
				where = "scenes.script_published > '0001-01-01 00:00:00+00:00'"
			}
		case "Has AI Generated Script":
			// querying the scenes in from alternate sources (stored in external_reference) has a performance impact, so it's user choice
			if config.Advanced.UseAltSrcInFileMatching {
				where = "(scenes.ai_script = 1 or (select distinct 1 from external_reference_links erl join external_references er on er.id=erl.external_reference_id where erl.internal_table='scenes' and internal_db_id=scenes.id and JSON_EXTRACT(er.external_data, '$.scene.ai_script') = 1))"
			} else {
				where = "scenes.ai_script = 1"
			}
		case "Has Human Generated Script":
			// querying the scenes in from alternate sources (stored in external_reference) has a performance impact, so it's user choice
			if config.Advanced.UseAltSrcInFileMatching {
				where = "(scenes.human_script = 1 or (select distinct 1 from external_reference_links erl join external_references er on er.id=erl.external_reference_id where erl.internal_table='scenes' and internal_db_id=scenes.id and JSON_EXTRACT(er.external_data, '$.scene.human_script') = 1))"
			} else {
				where = "scenes.human_script = 1"
			}
		case "Has Favourite Actor":
			where = "exists (select * from scene_cast join actors on actors.id=scene_cast.actor_id where actors.favourite=1 and scene_cast.scene_id=scenes.id)"
		case "Has Actor in Watchlist":
			where = "exists (select * from scene_cast join actors on actors.id=scene_cast.actor_id where actors.watchlist=1 and scene_cast.scene_id=scenes.id)"
		case "Available from POVR":
			where = "exists (select 1 from external_reference_links where external_source like 'alternate scene %' and external_id like 'povr-%' and internal_db_id = scenes.id)"
		case "Available from VRPorn":
			where = "exists (select 1 from external_reference_links where external_source like 'alternate scene %' and external_id like 'vrporn-%' and internal_db_id = scenes.id)"
		case "Available from SLR":
			where = "exists (select 1 from external_reference_links where external_source like 'alternate scene %' and external_id like 'slr-%' and internal_db_id = scenes.id)"
		case "Available from Alternate Sites":
			where = "exists (select 1 from external_reference_links where external_source like 'alternate scene %' and internal_db_id = scenes.id)"
		case "Multiple Scenes Available at an Alternate Site":
			where = "exists (select 1 from external_reference_links where external_source like 'alternate scene %' and internal_db_id = scenes.id  group by external_source having count(*)>1)"
		}

		if negate {
			where = "not " + where
		}

		if negate || mustHave {
			andAttribute = append(andAttribute, where)
		} else {
			orAttribute = append(orAttribute, where)
		}
	}

	if len(orAttribute) > 0 {
		combinedWhere = "(" + strings.Join(orAttribute, " or ") + ")"
	}
	if len(andAttribute) > 0 {
		if combinedWhere == "" {
			combinedWhere = strings.Join(andAttribute, " and ")
		} else {
			combinedWhere = combinedWhere + " and " + strings.Join(andAttribute, " and ")
		}
	}
	tx = tx.Where(combinedWhere)

	var sites []string
	var excludedSites []string
	for _, i := range r.Sites {
		switch firstchar := string(i.OrElse(" ")[0]); firstchar {
		case "!":
			exSite, _ := i.Get()
			excludedSites = append(excludedSites, exSite[1:])
		default:
			sites = append(sites, i.OrElse(""))
		}
	}

	if len(sites) > 0 {
		tx = tx.Where("site IN (?)", sites)
	}
	for _, exclude := range excludedSites {
		tx = tx.Where("site NOT IN (?)", exclude)
	}

	var tags []string
	var excludedTags []string
	var mustHaveTags []string
	for _, i := range r.Tags {
		switch firstchar := string(i.OrElse(" ")[0]); firstchar {
		case "&":
			inclTag, _ := i.Get()
			mustHaveTags = append(mustHaveTags, inclTag[1:])
		case "!":
			exTag, _ := i.Get()
			excludedTags = append(excludedTags, exTag[1:])
		default:
			tags = append(tags, i.OrElse(""))
		}
	}
	if len(tags) > 0 {
		tx = tx.
			Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
			Joins("left join tags on tags.id=scene_tags.tag_id").
			Where("tags.name IN (?)", tags)
	}
	for idx, musthave := range mustHaveTags {
		stAlias := "st_i" + strconv.Itoa(idx)
		tagAlias := "t_i" + strconv.Itoa(idx)
		tx = tx.
			Joins("join scene_tags "+stAlias+" on "+stAlias+".scene_id=scenes.id").
			Joins("join tags "+tagAlias+" on "+tagAlias+".id="+stAlias+".tag_id and "+tagAlias+".name=?", musthave)
	}
	for idx, exclude := range excludedTags {
		stAlias := "st_e" + strconv.Itoa(idx)
		tagAlias := "t_e" + strconv.Itoa(idx)
		tx = tx.Where("scenes.id not in (select "+stAlias+".scene_id  from tags "+tagAlias+" join scene_tags "+stAlias+" on "+stAlias+".scene_id =scenes.id and "+tagAlias+".id ="+stAlias+".tag_id where "+tagAlias+".name =?)", exclude)
	}

	var cast []string
	var mustHaveCast []string
	var excludedCast []string
	for _, i := range r.Cast {
		switch firstchar := string(i.OrElse(" ")[0]); firstchar {
		case "&":
			inclCast, _ := i.Get()
			mustHaveCast = append(mustHaveCast, inclCast[1:])
		case "!":
			exCast, _ := i.Get()
			excludedCast = append(excludedCast, exCast[1:])
		default:
			cast = append(cast, i.OrElse(""))
		}
	}
	if len(cast) > 0 {
		tx = tx.
			Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
			Joins("left join actors on actors.id=scene_cast.actor_id").
			Where("actors.name IN (?)", cast)
	}
	for idx, musthave := range mustHaveCast {
		scAlias := "sc_i" + strconv.Itoa(idx)
		actorAlias := "a_i" + strconv.Itoa(idx)
		tx = tx.
			Joins("join scene_cast "+scAlias+" on "+scAlias+".scene_id=scenes.id").
			Joins("join actors "+actorAlias+" on "+actorAlias+".id="+scAlias+".actor_id and "+actorAlias+".name=?", musthave)
	}
	for idx, exclude := range excludedCast {
		scAlias := "sc_e" + strconv.Itoa(idx)
		actorAlias := "a_e" + strconv.Itoa(idx)
		tx = tx.Where("scenes.id not in (select "+scAlias+".scene_id  from actors "+actorAlias+" join scene_cast "+scAlias+" on "+scAlias+".scene_id =scenes.id and "+actorAlias+".id ="+scAlias+".actor_id where "+actorAlias+".name =?)", exclude)
	}

	var cuepoint []string
	var mustHaveCuepoint []string
	var excludedCuepoint []string
	for _, i := range r.Cuepoint {
		cp := i.OrElse(" ")

		switch firstchar := cp[:1]; firstchar {
		case "&":
			mustHaveCuepoint = append(mustHaveCuepoint, setCuepointString(cp[1:]))
		case "!":
			excludedCuepoint = append(excludedCuepoint, setCuepointString(cp[1:]))
		default:
			cuepoint = append(cuepoint, setCuepointString(cp))
		}
	}

	if len(cuepoint) > 0 {
		tx = tx.Joins("left join scene_cuepoints on scene_cuepoints.scene_id=scenes.id")
		fields := []string{}
		values := []interface{}{}

		for _, i := range cuepoint {
			fields = append(fields, "scene_cuepoints.name LIKE ?")
			values = append(values, i)
		}
		tx = tx.Where(strings.Join(fields, " OR "), values...)
	}
	for idx, musthave := range mustHaveCuepoint {
		scpAlias := "scp_i" + strconv.Itoa(idx)
		if musthave == "%" {
			tx = tx.Where("scenes.id in (select case when count(*)>0 then scenes.id else null end from scene_cuepoints " + scpAlias + " where " + scpAlias + ".scene_id = scenes.id)")
		} else {
			tx = tx.Joins("join scene_cuepoints "+scpAlias+" on "+scpAlias+".scene_id=scenes.id and "+scpAlias+".name like ?", musthave)
		}
	}
	for idx, exclude := range excludedCuepoint {
		scpAlias := "scp_e" + strconv.Itoa(idx)
		if exclude == "%" {
			tx = tx.Where("scenes.id in (select case when count(*)=0 then scenes.id else null end from scene_cuepoints " + scpAlias + " where " + scpAlias + ".scene_id = scenes.id)")
		} else {
			tx = tx.Where("scenes.id not in (select "+scpAlias+".scene_id  from scene_cuepoints "+scpAlias+" where "+scpAlias+".scene_id =scenes.id and "+scpAlias+".name like ?)", exclude)
		}
	}

	if r.Released.Present() {
		tx = tx.Where("release_date_text LIKE ?", r.Released.OrElse("")+"%")
	}

	switch r.Sort.OrElse("") {
	case "added_desc":
		tx = tx.Order("added_date desc")
	case "added_asc":
		tx = tx.Order("added_date asc")
	case "release_desc":
		tx = tx.Order("release_date desc")
	case "release_asc":
		tx = tx.Order("release_date asc")
	case "title_desc":
		switch tx.Dialect().GetName() {
		case "mysql":
			tx = tx.Order("title desc")
		case "sqlite3":
			tx = tx.Order("title COLLATE NOCASE desc")
		}
	case "title_asc":
		switch tx.Dialect().GetName() {
		case "mysql":
			tx = tx.Order("title asc")
		case "sqlite3":
			tx = tx.Order("title COLLATE NOCASE asc")
		}
	case "total_file_size_desc":
		tx = tx.Order("total_file_size desc")
	case "total_file_size_asc":
		tx = tx.Order("total_file_size asc")
	case "total_watch_time_desc":
		tx = tx.Order("total_watch_time desc")
	case "total_watch_time_asc":
		tx = tx.Order("total_watch_time asc")
	case "rating_desc":
		tx = tx.
			Where("scenes.star_rating > 0").
			Order("scenes.star_rating desc")
	case "rating_asc":
		tx = tx.
			Where("scenes.star_rating > 0").
			Order("scenes.star_rating asc")
	case "last_opened_desc":
		tx = tx.
			Where("last_opened > '0001-01-01 00:00:00+00:00'").
			Order("last_opened desc")
	case "last_opened_asc":
		tx = tx.
			Where("last_opened > '0001-01-01 00:00:00+00:00'").
			Order("last_opened asc")
	case "scene_added_desc":
		tx = tx.Order("created_at desc")
	case "scene_updated_desc":
		tx = tx.Order("updated_at desc")
	case "script_published_desc":
		// querying the scenes in from alternate sources (stored in external_reference) has a performance impact, so it's user choice
		if config.Advanced.UseAltSrcInFileMatching {
			tx = tx.Order(`
						case when script_published > (select max(er.udf_datetime1) from external_reference_links erl join external_references er on er.id=erl.external_reference_id where erl.internal_table='scenes' and erl.internal_db_id=scenes.id and er.external_source like 'alternate scene %')
						then script_published
						else (select max(er.udf_datetime1) from external_reference_links erl join external_references er on er.id=erl.external_reference_id where erl.internal_table='scenes' and erl.internal_db_id=scenes.id and er.external_source like 'alternate scene %')
						end desc`)
		} else {
			tx = tx.Order("script_published desc")
		}
	case "scene_id_desc":
		tx = tx.Order("scene_id desc")
	case "site_asc":
		tx = tx.Order("scenes.site")
	case "random":
		if dbConn.Driver == "mysql" {
			tx = tx.Order("rand()")
		} else {
			tx = tx.Order("random()")
		}
	case "alt_src_desc":
		//tx = tx.Order(`(select max(er.external_date) from external_reference_links erl join external_references er on er.id=erl.external_reference_id where erl.internal_table='scenes' and erl.internal_db_id=scenes.id and er.external_source like 'alternate scene %') desc`)
		tx = tx.Order(`(select max(erl.udf_datetime1) from external_reference_links erl where erl.internal_table='scenes' and erl.internal_db_id=scenes.id and erl.external_source like 'alternate scene %') desc`)
	default:
		tx = tx.Order("release_date desc")
	}

	// Add second order to keep things stable in case of ties
	tx = tx.Order("scenes.id asc")

	preCountTx := tx.Group("scenes.scene_id")
	tx = tx.Group("scenes.scene_id")

	// Apply avail/accessible after counting
	if r.IsAvailable.Present() {
		tx = tx.Where("is_available = ?", r.IsAvailable.OrElse(true))
	}

	if r.IsAccessible.Present() {
		tx = tx.Where("is_accessible = ?", r.IsAccessible.OrElse(true))
	}

	if r.DlState.OrElse("") == "hidden" {
		tx = tx.Where("is_hidden = ?", true)
	} else {
		tx = tx.Where("is_hidden = ?", false)
	}

	// Pagination
	if r.Limit.Present() {
		tx = tx.Limit(r.Limit.MustGet()).Offset(r.Offset.OrElse(0))
	}

	return preCountTx, tx
}

func setCuepointString(cuepoint string) string {
	// swap * wildcard to sql wildcard %
	cuepoint = strings.Replace(cuepoint, "*", "%", -1)

	// if wrapped in quotes don't use wildcards
	if strings.HasPrefix(cuepoint, "\"") && strings.HasSuffix(cuepoint, "\"") {
		return cuepoint[1 : len(cuepoint)-1]
	} else {
		return "%" + cuepoint + "%"
	}
}

func getConfig(db *gorm.DB) Config {
	var config Config

	var kv KV
	db.Where("`key`='config'").First(&kv)

	json.Unmarshal([]byte(kv.Value), &config)
	return config
}
