package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/avast/retry-go"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/common"
)

// SceneCuepoint data model
type SceneCuepoint struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	SceneID   uint    `json:"-"`
	TimeStart float64 `json:"time_start"`
	Name      string  `json:"name"`
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
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	SceneID         string    `json:"scene_id"`
	Title           string    `json:"title"`
	SceneType       string    `json:"scene_type"`
	Studio          string    `json:"studio"`
	Site            string    `json:"site"`
	Tags            []Tag     `gorm:"many2many:scene_tags;" json:"tags"`
	Cast            []Actor   `gorm:"many2many:scene_cast;" json:"cast"`
	FilenamesArr    string    `json:"filenames_arr" sql:"type:text;"`
	Images          string    `json:"images" sql:"type:text;"`
	Files           []File    `json:"file"`
	Duration        int       `json:"duration"`
	Synopsis        string    `json:"synopsis" sql:"type:text;"`
	ReleaseDate     time.Time `json:"release_date"`
	ReleaseDateText string    `json:"release_date_text"`
	CoverURL        string    `json:"cover_url"`
	SceneURL        string    `json:"scene_url"`
	IsMultipart     bool      `json:"is_multipart"`

	StarRating     float64         `json:"star_rating"`
	Favourite      bool            `json:"favourite" gorm:"default:false"`
	Watchlist      bool            `json:"watchlist" gorm:"default:false"`
	IsAvailable    bool            `json:"is_available" gorm:"default:false"`
	IsAccessible   bool            `json:"is_accessible" gorm:"default:false"`
	IsWatched      bool            `json:"is_watched" gorm:"default:false"`
	IsScripted     bool            `json:"is_scripted" gorm:"default:false"`
	Cuepoints      []SceneCuepoint `json:"cuepoints"`
	History        []History       `json:"history"`
	AddedDate      time.Time       `json:"added_date"`
	LastOpened     time.Time       `json:"last_opened"`
	TotalFileSize  int64           `json:"total_file_size"`
	TotalWatchTime int             `json:"total_watch_time" gorm:"default:0"`

	HasVideoPreview bool `json:"has_preview" gorm:"default:false"`
	// HasVideoThumbnail bool `json:"has_video_thumbnail" gorm:"default:false"`

	NeedsUpdate bool `json:"needs_update"`

	Fulltext string  `gorm:"-" json:"fulltext"`
	Score    float64 `gorm:"-" json:"_score"`
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
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "video").Find(&files)

	return files, nil
}

func (o *Scene) GetScriptFiles() ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Preload("Volume").Where("scene_id = ? AND type = ?", o.ID, "script").Order("is_selected_script DESC, created_time DESC").Find(&files)

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
	var o Scene
	db.Where(&Scene{SceneID: ext.SceneID}).FirstOrCreate(&o)

	o.NeedsUpdate = false
	o.SceneID = ext.SceneID

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
		images = append(images, Image{
			URL:  ext.Covers[i],
			Type: "cover",
		})
	}

	for i := range ext.Gallery {
		images = append(images, Image{
			URL:  ext.Gallery[i],
			Type: "gallery",
		})
	}

	imgTxt, err := json.Marshal(images)
	if err == nil {
		o.Images = string(imgTxt)
	}

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
		db.Model(&o).Association("Cast").Append(tmpActor)
	}

	return nil
}

type RequestSceneList struct {
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
			tx = tx.Where("watchlist = ?", true)
		}
		if i.OrElse("") == "favourite" {
			tx = tx.Where("favourite = ?", true)
		}
		if i.OrElse("") == "scripted" {
			tx = tx.Where("is_scripted = ?", true)
		}
	}

	var sites []string
	for _, i := range r.Sites {
		sites = append(sites, i.OrElse(""))
	}
	if len(sites) > 0 {
		tx = tx.Where("site IN (?)", sites)
	}

	var tags []string
	for _, i := range r.Tags {
		tags = append(tags, i.OrElse(""))
	}
	if len(tags) > 0 {
		tx = tx.
			Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
			Joins("left join tags on tags.id=scene_tags.tag_id").
			Where("tags.name IN (?)", tags)
	}

	var cast []string
	for _, i := range r.Cast {
		cast = append(cast, i.OrElse(""))
	}
	if len(cast) > 0 {
		tx = tx.
			Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
			Joins("left join actors on actors.id=scene_cast.actor_id").
			Where("actors.name IN (?)", cast)
	}

	var cuepoint []string
	for _, i := range r.Cuepoint {
		cuepoint = append(cuepoint, i.OrElse(""))
	}
	if len(cuepoint) > 0 {
		tx = tx.Joins("left join scene_cuepoints on scene_cuepoints.scene_id=scenes.id")
		for _, i := range cuepoint {
			tx = tx.Where("scene_cuepoints.name LIKE ?", "%"+i+"%")
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
			Where("star_rating > ?", 0).
			Order("star_rating desc")
	case "rating_asc":
		tx = tx.
			Where("star_rating > ?", 0).
			Order("star_rating asc")
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
	tx.Group("scenes.scene_id").Count(&out.CountAny)
	tx.Group("scenes.scene_id").Where("is_available = ?", true).Where("is_accessible = ?", true).Count(&out.CountAvailable)
	tx.Group("scenes.scene_id").Where("is_available = ?", true).Count(&out.CountDownloaded)
	tx.Group("scenes.scene_id").Where("is_available = ?", false).Count(&out.CountNotDownloaded)

	// Apply avail/accessible after counting
	if r.IsAvailable.Present() {
		tx = tx.Where("is_available = ?", r.IsAvailable.OrElse(true))
	}

	if r.IsAccessible.Present() {
		tx = tx.Where("is_accessible = ?", r.IsAccessible.OrElse(true))
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
