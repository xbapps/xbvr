package xbase

import (
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

type PossibleFilename struct {
	ID     uint    `gorm:"primary_key" json:"id"`
	Name   string  `gorm:"unique_index" json:"name"`
	Scenes []Scene `gorm:"many2many:scene_filenames;" json:"scenes"`
}

type Scene struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	SceneID         string             `json:"scene_id"`
	Title           string             `json:"title"`
	SceneType       string             `json:"scene_type"`
	Studio          string             `json:"studio"`
	Site            string             `json:"site"`
	Tags            []Tag              `gorm:"many2many:scene_tags;" json:"tags"`
	Cast            []Actor            `gorm:"many2many:scene_cast;" json:"cast"`
	Filenames       []PossibleFilename `gorm:"many2many:scene_filenames;" json:"filename"`
	Images          []Image            `json:"images"`
	Files           []File             `json:"file"`
	Duration        int                `json:"duration"`
	Synopsis        string             `json:"synopsis" sql:"type:text;"`
	ReleaseDate     time.Time          `json:"release_date"`
	ReleaseDateText string             `json:"release_date_text"`
	CoverURL        string             `json:"cover_url"`
	SceneURL        string             `json:"scene_url"`
	Rating          int                `json:"rating"`
	Favourite       bool               `json:"favourite"`
	Wishlist        bool               `json:"wishlist"`
	Watchlist       bool               `json:"watchlist"`
	IsAvailable     bool               `json:"is_available"`
	IsAccessible    bool               `json:"is_accessible"`
}

func (i *Scene) Save() error {
	db, _ := GetDB()
	err := db.Save(i).Error
	db.Close()
	return err
}

func (o *Scene) GetIfExist(id string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Preload("Tags").Preload("Cast").Preload("Filenames").Where(&Scene{SceneID: id}).First(o).Error
}

func (o *Scene) GetIfExistURL(u string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Preload("Tags").Preload("Cast").Preload("Filenames").Where(&Scene{SceneURL: u}).First(o).Error
}

func (o *Scene) GetFiles() ([]File, error) {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	db.Where(&File{SceneID: o.ID}).Find(&files)

	return files, nil
}

func (o *Scene) CreateUpdateFromExternal(ext ExtScene) error {
	// Check if scene exist
	// o.GetIfExist(ext.SceneID)

	// Save
	db, _ := GetDB()
	defer db.Close()

	db.Where(&Scene{SceneID: ext.SceneID}).FirstOrCreate(&o)

	o.SceneID = ext.SceneID
	o.Title = ext.Title
	o.SceneType = ext.SceneType
	o.Studio = ext.Studio
	o.Site = ext.Site
	o.Duration = ext.Duration
	o.Synopsis = ext.Synopsis
	o.ReleaseDateText = ext.Released
	o.CoverURL = ext.Covers[0]
	o.SceneURL = ext.HomepageURL

	if ext.Released != "" {
		dateParsed, err := dateparse.ParseLocal(strings.Replace(ext.Released, ",", "", -1))
		if err == nil {
			o.ReleaseDate = dateParsed
		}
	}

	db.Save(o)

	// Clean & Associate Tags
	var tmpTag Tag
	for _, name := range ext.Tags {
		tagClean := convertTag(name)
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

	// Associate Filenames
	var tmpSceneFilename PossibleFilename
	for _, name := range ext.Filenames {
		tmpSceneFilename = PossibleFilename{}
		db.Where(&PossibleFilename{Name: name}).FirstOrCreate(&tmpSceneFilename)
		db.Model(&o).Association("Filenames").Append(tmpSceneFilename)
	}

	// Associate Images (but first remove old ones)
	db.Unscoped().Where(&Image{SceneID: o.ID}).Delete(Image{})

	for _, u := range ext.Covers {
		tmpImage := Image{}
		db.Where(&Image{URL: u}).FirstOrCreate(&tmpImage)
		tmpImage.SceneID = o.ID
		tmpImage.Type = "cover"
		tmpImage.Save()
	}

	for _, u := range ext.Gallery {
		tmpImage := Image{}
		db.Where(&Image{URL: u}).FirstOrCreate(&tmpImage)
		tmpImage.SceneID = o.ID
		tmpImage.Type = "gallery"
		tmpImage.Save()
	}

	return nil
}
