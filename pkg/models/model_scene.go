package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/jinzhu/gorm"
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
	err := db.Save(o).Error
	db.Close()
	return err
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

	StarRating   float64         `json:"star_rating"`
	Favourite    bool            `json:"favourite" gorm:"default:false"`
	Watchlist    bool            `json:"watchlist" gorm:"default:false"`
	IsAvailable  bool            `json:"is_available" gorm:"default:false"`
	IsAccessible bool            `json:"is_accessible" gorm:"default:false"`
	IsWatched    bool            `json:"is_watched" gorm:"default:false"`
	Cuepoints    []SceneCuepoint `json:"cuepoints"`
	History      []History       `json:"history"`
	AddedDate    time.Time       `json:"added_date"`
	LastOpened   time.Time       `json:"last_opened"`

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
	err := db.Save(i).Error
	db.Close()
	return err
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

func (o *Scene) GetFiles() ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Preload("Volume").Where(&File{SceneID: o.ID}).Find(&files)

	return files, nil
}

func (o *Scene) UpdateStatus() {
	// Check if file with scene association exists
	files, err := o.GetFiles()
	if err != nil {
		return
	}

	changed := false

	if len(files) > 0 {
		if !o.IsAvailable {
			o.IsAvailable = true
			changed = true
		}

		var newestFileDate time.Time
		for j := range files {
			if files[j].Exists() {
				if files[j].CreatedTime.Before(newestFileDate) || newestFileDate.IsZero() {
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

		if !newestFileDate.Equal(o.AddedDate) && !newestFileDate.IsZero() {
			o.AddedDate = newestFileDate
			changed = true
		}
	} else {
		if o.IsAvailable {
			o.IsAvailable = false
			changed = true
		}
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

	db.Save(o)

	// Clean & Associate Tags
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
	var tmpActor Actor
	for _, name := range ext.Cast {
		tmpActor = Actor{}
		db.Where(&Actor{Name: strings.Replace(name, ".", "", -1)}).FirstOrCreate(&tmpActor)
		db.Model(&o).Association("Cast").Append(tmpActor)
	}

	return nil
}
