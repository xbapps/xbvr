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
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&o).Error
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

	SceneID         string    `json:"scene_id" xbvrbackup:"scene_id"`
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
	CoverURL        string    `json:"cover_url" xbvrbackup:"cover_url"`
	SceneURL        string    `json:"scene_url" xbvrbackup:"scene_url"`
	MemberURL       string    `json:"members_url" xbvrbackup:"members_url"`
	IsMultipart     bool      `json:"is_multipart" xbvrbackup:"is_multipart"`

	StarRating     float64         `json:"star_rating" xbvrbackup:"star_rating"`
	Favourite      bool            `json:"favourite" gorm:"default:false" xbvrbackup:"favourite"`
	Watchlist      bool            `json:"watchlist" gorm:"default:false" xbvrbackup:"watchlist"`
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
	TrailerSource string `gorm:"size:1000" json:"trailer_source" xbvrbackup:"trailer_source"`
	ChromaKey     string `json:"passthrough" xbvrbackup:"passthrough"`
	Trailerlist   bool   `json:"trailerlist" gorm:"default:false" xbvrbackup:"trailerlist"`
	IsSubscribed  bool   `json:"is_subscribed" gorm:"default:false"`
	IsHidden      bool   `json:"is_hidden" gorm:"default:false" xbvrbackup:"is_hidden"`
	LegacySceneID string `json:"legacy_scene_id" xbvrbackup:"legacy_scene_id"`

	Description string  `gorm:"-" json:"description" xbvrbackup:"-"`
	Score       float64 `gorm:"-" json:"_score" xbvrbackup:"-"`
}

type Image struct {
	URL         string `json:"url"`
	Type        string `json:"type"`
	Orientation string `json:"orientation"`
}

func (i *Scene) Save() error {
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&i).Error
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
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Tags").
		Preload("Cast").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints").
		Where(&Scene{SceneID: id}).First(o).Error
}

func (o *Scene) GetIfExistByPK(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Tags").
		Preload("Cast").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints").
		Where(&Scene{ID: id}).First(o).Error
}

func (o *Scene) GetIfExistURL(u string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.
		Preload("Tags").
		Preload("Cast").
		Preload("Files").
		Preload("History").
		Preload("Cuepoints").
		Where(&Scene{SceneURL: u}).First(o).Error
}

func (o *Scene) GetFunscriptTitle() string {

	// first make the title filename safe
	var re = regexp.MustCompile(`[?/\<>|]`)

	title := o.Title
	// Colons are pretty common in titles, so we use a unicode alternative
	title = strings.ReplaceAll(title, ":", "꞉")
	// all other unsafe characters get removed
	title = re.ReplaceAllString(title, "")

	// add ID to prevent title collisions
	return fmt.Sprintf("%d - %s", o.ID, title)
}

func (o *Scene) GetFiles() ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Preload("Volume").Where(&File{SceneID: o.ID}).Find(&files)

	return files, nil
}

func (o *Scene) GetTotalWatchTime() int {
	db, _ := GetDB()
	defer db.Close()

	totalResult := struct{ Total float64 }{}
	db.Raw(`select sum(duration) as total from histories where scene_id = ?`, o.ID).Scan(&totalResult)

	return int(totalResult.Total)
}

func (o *Scene) GetVideoFiles() ([]File, error) {
	files, err := o.GetVideoFilesSorted("")
	return files, err
}
func (o *Scene) GetVideoFilesSorted(sort string) ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	if sort == "" {
		db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "video").Find(&files)
	} else {
		db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "video").Order(sort).Find(&files)
	}

	return files, nil
}

func (o *Scene) GetScriptFiles() ([]File, error) {
	var files, err = o.GetScriptFilesSorted("is_selected_script DESC, created_time DESC")
	return files, err
}
func (o *Scene) GetScriptFilesSorted(sort string) ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	if sort == "" {
		db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "script").Find(&files)
	} else {
		db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "script").Order(sort).Find(&files)
	}

	return files, nil
}

func (o *Scene) GetHSPFiles() ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "hsp").Find(&files)

	return files, nil
}

func (o *Scene) GetSubtitlesFiles() ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "subtitles").Find(&files)

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
					if files[j].CreatedTime.After(newestFileDate) || newestFileDate.IsZero() {
						newestFileDate = files[j].CreatedTime
					}
					if !o.IsAccessible {
						o.IsAccessible = true
						changed = true
					}
				} else {
					if o.IsAccessible {
						o.IsAccessible = false
						changed = true
					}
				}
			}
		}

		if totalFileSize != o.TotalFileSize {
			o.TotalFileSize = totalFileSize
			changed = true
		}

		if scripts > 0 && o.IsScripted == false {
			o.IsScripted = true
			changed = true
		}

		if scripts == 0 && o.IsScripted == true {
			o.IsScripted = false
			changed = true
		}

		if videos > 0 && o.IsAvailable == false {
			o.IsAvailable = true
			changed = true
		}

		if videos == 0 && o.IsAvailable == true {
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

		if o.IsScripted == true {
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

	o.NeedsUpdate = false
	o.EditsApplied = false
	o.SceneID = ext.SceneID
	o.ScraperId = ext.ScraperID

	if o.Title != ext.Title {
		o.Title = ext.Title

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
	SaveWithRetry(db, &o)

	// Clean & Associate Tags
	db.Model(&o).Association("Tags").Clear()
	var tmpTag Tag
	for _, name := range ext.Tags {
		tagClean := ConvertTag(name)
		if tagClean != "" {
			tmpTag = Tag{}
			db.Where(&Tag{Name: tagClean}).FirstOrCreate(&tmpTag)
			db.Model(&o).Association("Tags").Append(tmpTag)
		}
	}

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
			//AddActionActor(name, ext.ActorDetails[name].Source, "add", "image_url", ext.ActorDetails[name].ImageUrl)
		}
		if ext.ActorDetails[name].ProfileUrl != "" {
			if tmpActor.AddToActorUrlArray(ActorLink{Url: ext.ActorDetails[name].ProfileUrl, Type: ext.ActorDetails[name].Source}) {
				saveActor = true
			}
			//AddActionActor(name, ext.ActorDetails[name].Source, "add", "image_url", ext.ActorDetails[name].ImageUrl)
		}
		if saveActor {
			tmpActor.Save()
		}
		db.Model(&o).Association("Cast").Append(tmpActor)
	}

	return nil
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
	limit := r.Limit.OrElse(100)
	offset := r.Offset.OrElse(0)

	db, _ := GetDB()
	defer db.Close()

	var scenes []Scene
	tx := db.Model(&scenes)

	var out ResponseSceneList

	if enablePreload {
		tx = tx.
			Preload("Cast").
			Preload("Tags").
			Preload("Files").
			Preload("History").
			Preload("Cuepoints")
	}

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
		if i.OrElse("") == "scripted" {
			tx = tx.Where("is_scripted = ?", true)
		}
	}

	// handle Attribute selections
	var orAttribute []string
	var andAttribute []string
	combinedWhere := ""
	for idx, attribute := range r.Attributes {
		truefalse := true
		fieldName := attribute.OrElse("")
		sceneAlias := "scenes_f" + strconv.Itoa(idx)
		fileAlias := "files_f" + strconv.Itoa(idx)
		scenecastAlias := "scene_cast_f" + strconv.Itoa(idx)
		actorsAlias := "actors_f" + strconv.Itoa(idx)
		scenecuepointAlias := "scene_cuepoints_f" + strconv.Itoa(idx)
		erlAlias := "external_reference_links_f" + strconv.Itoa(idx)

		if strings.HasPrefix(fieldName, "!") { // ! prefix indicate NOT filtering
			truefalse = false
			fieldName = fieldName[1:]
		}
		if strings.HasPrefix(fieldName, "&") { // & prefix indicate must have filtering
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
		where := ""
		switch fieldName {
		case "Multiple Video Files":
			if truefalse {
				where = "scenes.id in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'video' group by " + fileAlias + ".scene_id having count(*) >1)"
			} else {
				where = "scenes.id not in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'video' group by " + fileAlias + ".scene_id having count(*) >1)"
			}
		case "Single Video File":
			if truefalse {
				where = "scenes.id in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'video' group by " + fileAlias + ".scene_id having count(*) =1)"
			} else {
				where = "scenes.id not in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'video' group by " + fileAlias + ".scene_id having count(*) =1)"
			}
		case "Multiple Script Files":
			if truefalse {
				where = "scenes.id in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'script' group by " + fileAlias + ".scene_id having count(*) >1)"
			} else {
				where = "scenes.id not in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'script' group by " + fileAlias + ".scene_id having count(*) >1)"
			}
		case "Single Script File":
			if truefalse {
				where = "scenes.id in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'script' group by " + fileAlias + ".scene_id having count(*) =1)"
			} else {
				where = "scenes.id not in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'script' group by " + fileAlias + ".scene_id having count(*) =1)"
			}
		case "Has Hsp File":
			if truefalse {
				where = "scenes.id in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'hsp' group by " + fileAlias + ".scene_id having count(*) >0)"
			} else {
				where = "scenes.id not in (select " + sceneAlias + ".id from scenes " + sceneAlias + " join files " + fileAlias + " on " + fileAlias + ".scene_id = " + sceneAlias + ".id and " + fileAlias + ".`type` = 'hsp' where " + sceneAlias + ".id=scenes.id group by " + sceneAlias + ".id)"
			}
		case "Has Subtitles File":
			if truefalse {
				where = "scenes.id in (select " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".`type` = 'subtitles' group by " + fileAlias + ".scene_id having count(*) >0)"
			} else {
				where = "scenes.id not in (select " + sceneAlias + ".id from scenes " + sceneAlias + " join files " + fileAlias + " on " + fileAlias + ".scene_id = " + sceneAlias + ".id and " + fileAlias + ".`type` = 'subtitles' where " + sceneAlias + ".id=scenes.id group by " + sceneAlias + ".id)"
			}
		case "Has Rating":
			if truefalse {
				where = "scenes.star_rating > 0"
			} else {
				where = "scenes.star_rating = 0"
			}
		case "Has Cuepoints":
			if truefalse {
				where = "scenes.id in (select " + scenecuepointAlias + ".scene_id from scene_cuepoints " + scenecuepointAlias + " where " + scenecuepointAlias + ".scene_id =scenes.id)"
			} else {
				where = "scenes.id not in (select " + scenecuepointAlias + ".scene_id from scene_cuepoints " + scenecuepointAlias + " where " + scenecuepointAlias + ".scene_id =scenes.id)"
			}
		case "Has Simple Cuepoints":
			if truefalse {
				where = "scenes.id in (select " + scenecuepointAlias + ".scene_id from scene_cuepoints " + scenecuepointAlias + " where " + scenecuepointAlias + ".scene_id =scenes.id and track is null)"
			} else {
				where = "scenes.id not in (select " + scenecuepointAlias + ".scene_id from scene_cuepoints " + scenecuepointAlias + " where " + scenecuepointAlias + ".scene_id =scenes.id and track is null)"
			}
		case "Has HSP Cuepoints":
			if truefalse {
				where = "scenes.id in (select " + scenecuepointAlias + ".scene_id from scene_cuepoints " + scenecuepointAlias + " where " + scenecuepointAlias + ".scene_id =scenes.id and track is not null)"
			} else {
				where = "scenes.id not in (select " + scenecuepointAlias + ".scene_id from scene_cuepoints " + scenecuepointAlias + " where " + scenecuepointAlias + ".scene_id =scenes.id and track is not null)"
			}
		case "In Trailer List":
			if truefalse {
				where = "trailerlist = 1"
			} else {
				where = "trailerlist = 0"
			}
		case "Has Subscription":
			if truefalse {
				where = "is_subscribed = 1"
			} else {
				where = "is_subscribed = 0"
			}
		case "Rating 0", "Rating .5", "Rating 1", "Rating 1.5", "Rating 2", "Rating 2.5", "Rating 3", "Rating 3.5", "Rating 4", "Rating 4.5", "Rating 5":
			if truefalse {
				where = "scenes.star_rating = " + fieldName[7:]
			} else {
				where = "scenes.star_rating <> " + fieldName[7:]
			}
		case "Cast 6+":
			if truefalse {
				where = "scenes.id in (select " + scenecastAlias + ".scene_id from scene_cast " + scenecastAlias + " join actors " + actorsAlias + " on " + actorsAlias + ".id =" + scenecastAlias + ".actor_id where " + scenecastAlias + ".scene_id =scenes.id and " + actorsAlias + ".name not like 'aka:%' group by " + scenecastAlias + ".scene_id having count(*)>5)"
			} else {
				where = "scenes.id not in (select " + scenecastAlias + ".scene_id from scene_cast " + scenecastAlias + " join actors " + actorsAlias + " on " + actorsAlias + ".id =" + scenecastAlias + ".actor_id where " + scenecastAlias + ".scene_id =scenes.id and " + actorsAlias + ".name not like 'aka:%' group by " + scenecastAlias + ".scene_id having count(*)>5)"
			}
		case "Cast 1", "Cast 2", "Cast 3", "Cast 4", "Cast 5":
			if truefalse {
				where = "scenes.id in (select " + scenecastAlias + ".scene_id from scene_cast " + scenecastAlias + " join actors " + actorsAlias + " on " + actorsAlias + ".id =" + scenecastAlias + ".actor_id where " + scenecastAlias + ".scene_id =scenes.id and " + actorsAlias + ".name not like 'aka:%' group by " + scenecastAlias + ".scene_id having count(*)=" + fieldName[5:] + ")"
			} else {
				where = "scenes.id not in (select " + scenecastAlias + ".scene_id from scene_cast " + scenecastAlias + " join actors " + actorsAlias + " on " + actorsAlias + ".id =" + scenecastAlias + ".actor_id where " + scenecastAlias + ".scene_id =scenes.id and " + actorsAlias + ".name not like 'aka:%' group by " + scenecastAlias + ".scene_id having count(*)=" + fieldName[5:] + ")"
			}
		case "Resolution":
			switch db.Dialect().GetName() {
			case "mysql":
				if truefalse {
					where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and case when " + fileAlias + ".video_projection = '360_tb' then (" + fileAlias + ".video_width+499)*2 div 1000 else (" + fileAlias + ".video_width+499) div 1000 end = " + value + ")"
				} else {
					where = "scenes.id not in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and case when " + fileAlias + ".video_projection = '360_tb' then (" + fileAlias + ".video_width+499)*2 div 1000 else (" + fileAlias + ".video_width+499) div 1000 end = " + value + ")"
				}
			default:
				if truefalse {
					where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and case when " + fileAlias + ".video_projection = '360_tb' then (" + fileAlias + ".video_width+499)*2 / 1000 else (" + fileAlias + ".video_width+499) / 1000 end = " + value + ")"
				} else {
					where = "scenes.id not in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and case when " + fileAlias + ".video_projection = '360_tb' then (" + fileAlias + ".video_width+499)*2 / 1000 else (" + fileAlias + ".video_width+499) / 1000 end = " + value + ")"
				}
			}
		case "Frame Rate":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_avg_frame_rate_val = " + value + " and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id not in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_avg_frame_rate_val = " + value + " and " + fileAlias + ".`type` = 'video')"
			}
		case "Flat video":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='flat' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='flat' and " + fileAlias + ".`type` = 'video')"
			}
		case "FOV: 180°":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('180_mono','180_sbs','fisheye') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('180_mono','180_sbs','fisheye') and " + fileAlias + ".`type` = 'video')"
			}
		case "FOV: 190°":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('rf52','fisheye190') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('rf52','fisheye190') and " + fileAlias + ".`type` = 'video')"
			}
		case "FOV: 200°":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='mkx200' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='mkx200' and " + fileAlias + ".`type` = 'video')"
			}
		case "FOV: 220°":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('mkx220','vrca220') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('mkx220','vrca220') and " + fileAlias + ".`type` = 'video')"
			}
		case "FOV: 360°":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('360_mono','360_tb') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('360_mono','360_tb') and " + fileAlias + ".`type` = 'video')"
			}
		case "Projection Perspective":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='flat' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='flat' and " + fileAlias + ".`type` = 'video')"
			}
		case "Projection Equirectangular":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('180_mono','180_sbs') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('180_mono','180_sbs') and " + fileAlias + ".`type` = 'video')"
			}
		case "Projection Equirectangular360":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('360_tb','360_mono') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('360_tb','360_mono') and " + fileAlias + ".`type` = 'video')"
			}
		case "Projection Fisheye":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('mkx200','mkx220','vrca220','rf52','fisheye190','fisheye') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('mkx200','mkx220','vrca220','rf52','fisheye190','fisheye') and " + fileAlias + ".`type` = 'video')"
			}
		case "Mono":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('flat','180_mono','360_mono') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('flat','180_mono','360_mono') and " + fileAlias + ".`type` = 'video')"
			}
		case "Top/Bottom":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='360_tb' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='360_tb' and " + fileAlias + ".`type` = 'video')"
			}
		case "Side by Side":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection not in ('360_tb','flat','180_mono','360_mono') and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection in ('360_tb','flat','180_mono','360_mono') and " + fileAlias + ".`type` = 'video')"
			}
		case "MKX200":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='mkx200' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='mkx200' and " + fileAlias + ".`type` = 'video')"
			}
		case "MKX220":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='mkx220' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='mkx220' and " + fileAlias + ".`type` = 'video')"
			}
		case "VRCA220":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='vrca220' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_projection ='vrca220' and " + fileAlias + ".`type` = 'video')"
			}
		case "Codec":
			if truefalse {
				where = "scenes.id in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_codec_name = '" + value + "' and " + fileAlias + ".`type` = 'video')"
			} else {
				where = "scenes.id not in (select distinct " + fileAlias + ".scene_id  from files " + fileAlias + " where " + fileAlias + ".scene_id = scenes.id and " + fileAlias + ".video_codec_name = '" + value + "' and " + fileAlias + ".`type` = 'video')"
			}
		case "In Watchlist":
			if truefalse {
				where = "scenes.watchlist = 1"
			} else {
				where = "scenes.watchlist = 0"
			}
		case "Is Scripted":
			if truefalse {
				where = "is_scripted = 1"
			} else {
				where = "is_scripted = 0"
			}
		case "Is Favourite":
			if truefalse {
				where = "scenes.favourite = 1"
			} else {
				where = "scenes.favourite = 0"
			}
		case "Stashdb Linked":
			if truefalse {
				where = "(select count(*) from external_reference_links " + erlAlias + " where " + erlAlias + ".internal_db_id = scenes.id and " + erlAlias + ".`external_source` = 'stashdb scene') > 0"
			} else {
				where = "(select count(*) from external_reference_links " + erlAlias + " where " + erlAlias + ".internal_db_id = scenes.id and " + erlAlias + ".`external_source` = 'stashdb scene') = 0"
			}
		case "POVR Scraper":
			if truefalse {
				where = `scenes.scene_id like "povr-%"`
			} else {
				where = `scenes.scene_id not like "povr-%"`
			}
		case "SLR Scraper":
			if truefalse {
				where = `scenes.scene_id like "slr-%"`
			} else {
				where = `scenes.scene_id not like "slr-%"`
			}
		case "VRPHub Scraper":
			if truefalse {
				where = `scenes.scene_id like "vrphub-%"`
			} else {
				where = `scenes.scene_id not like "vrphub-%"`
			}
		case "VRPorn Scraper":
			if truefalse {
				where = `scenes.scene_id like "vrporn-%"`
			} else {
				where = `scenes.scene_id not like "vrporn-%"`
			}
		}

		switch firstchar := string(attribute.OrElse(" ")[0]); firstchar {
		case "&", "!":
			andAttribute = append(andAttribute, where)
		default:
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
		switch db.Dialect().GetName() {
		case "mysql":
			tx = tx.Order("title desc")
		case "sqlite3":
			tx = tx.Order("title COLLATE NOCASE desc")
		}
	case "title_asc":
		switch db.Dialect().GetName() {
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
			Where("scenes.star_rating > ?", 0).
			Order("scenes.star_rating desc")
	case "rating_asc":
		tx = tx.
			Where("scenes.star_rating > ?", 0).
			Order("scenes.star_rating asc")
	case "last_opened_desc":
		tx = tx.
			Where("last_opened > ?", "0001-01-01 00:00:00+00:00").
			Order("last_opened desc")
	case "last_opened_asc":
		tx = tx.
			Where("last_opened > ?", "0001-01-01 00:00:00+00:00").
			Order("last_opened asc")
	case "scene_added_desc":
		tx = tx.Order("created_at desc")
	case "scene_updated_desc":
		tx = tx.Order("updated_at desc")
	case "random":
		if dbConn.Driver == "mysql" {
			tx = tx.Order("rand()")
		} else {
			tx = tx.Order("random()")
		}
	default:
		tx = tx.Order("release_date desc")
	}

	// Count other variations
	tx.Group("scenes.scene_id").Where("is_hidden = ?", false).Count(&out.CountAny)
	tx.Group("scenes.scene_id").Where("is_available = ?", true).Where("is_accessible = ?", true).Where("is_hidden = ?", false).Count(&out.CountAvailable)
	tx.Group("scenes.scene_id").Where("is_available = ?", true).Where("is_hidden = ?", false).Count(&out.CountDownloaded)
	tx.Group("scenes.scene_id").Where("is_available = ?", false).Where("is_hidden = ?", false).Count(&out.CountNotDownloaded)
	tx.Group("scenes.scene_id").Where("is_hidden = ?", true).Count(&out.CountHidden)

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

	// Count totals for selection
	tx.
		Group("scenes.scene_id").
		Count(&out.Results)

	// Get scenes
	tx.
		Group("scenes.scene_id").
		Limit(limit).
		Offset(offset).
		Find(&out.Scenes)

	return out
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
