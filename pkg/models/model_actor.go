package models

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Name         string  `gorm:"unique_index" json:"name"`
	Scenes       []Scene `gorm:"many2many:scene_cast;" json:"-"`
	ActorID      string  `json:"_id"`
	Aliases      *string `json:"aliases" sql:"type:text;"`
	Bio          string  `json:"bio"`
	Birthday     string  `json:"birthday"`
	Ethnicity    string  `json:"ethnicity"`
	EyeColor     string  `json:"eye_color"`
	Facebook     string  `json:"facebook"`
	HairColor    string  `json:"hair_color"`
	Height       int     `json:"height"`
	HomepageURL  string  `json:"homepage_url"`
	ImageURL     string  `json:"image_url"`
	Instagram    string  `json:"instagram"`
	Measurements string  `json:"measurements"`
	Nationality  string  `json:"nationality"`
	Reddit       string  `json:"reddit"`
	Twitter      string  `json:"twitter"`
	Weight       int     `json:"weight"`
	Count        int     `json:"count"`
}

func (i *Actor) Save() error {
	db, _ := GetDB()
	defer db.Close()

	err := db.Save(i).Error
	return err
}

func ResolveActorAliases() error {
	db, _ := GetDB()
	defer db.Close()

	// Get all actors with aliases
	var actors []Actor
	db.Model(Actor{}).Where("actors.aliases IS NOT NULL").Find(&actors)

	for _, actor := range actors {
		var dupes []Actor
		var aliases []string
		_ = json.Unmarshal([]byte(*actor.Aliases), &aliases)
		db.Model(Actor{}).Where("name IN (?)", aliases).Find(&dupes)
		if len(dupes) > 0 {
			for _, dupe := range dupes {
				if actor.Name != dupe.Name && dupe.ActorID == "" {
					var scenes []Scene
					db.Model(&scenes).Preload("Cast").
						Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
						Joins("left join actors on actors.id=scene_cast.actor_id").
						Where("actors.name = ?", dupe.Name).Find(&scenes)
					for _, scene := range scenes {
						db.Model(&scene).Association("Cast").Replace(actor)
						db.Delete(&dupe)
					}
				}
			}
		}
	}
	return nil
}

func ActorCreateUpdateFromExternal(db *gorm.DB, ext ScrapedActor) error {
	var o Actor
	db.Where(&Actor{Name: ext.Name}).FirstOrCreate(&o)

	if ext.ActorID != "" {
		o.ActorID = ext.ActorID
	}

	if ext.Aliases != "" {
		o.Aliases = &ext.Aliases
	}

	if ext.Bio != "" {
		o.Bio = ext.Bio
	}

	if ext.Birthday != "" {
		o.Birthday = ext.Birthday
	}

	if ext.Ethnicity != "" {
		o.Ethnicity = ext.Ethnicity
	}

	if ext.EyeColor != "" {
		o.EyeColor = ext.EyeColor
	}

	if ext.Facebook != "" {
		o.Facebook = ext.Facebook
	}

	if ext.HairColor != "" {
		o.HairColor = ext.HairColor
	}

	if ext.Height != 0 {
		o.Height = ext.Height
	}

	if ext.HomepageURL != "" {
		o.HomepageURL = ext.HomepageURL
	}

	if ext.ImageURL != "" {
		o.ImageURL = ext.ImageURL
	}

	if ext.Instagram != "" {
		o.Instagram = ext.Instagram
	}

	if ext.Measurements != "" {
		o.Measurements = ext.Measurements
	}

	if ext.Nationality != "" {
		o.Nationality = ext.Nationality
	}

	if ext.Reddit != "" {
		o.Reddit = ext.Reddit
	}

	if ext.Twitter != "" {
		o.Twitter = ext.Twitter
	}

	if ext.Weight != 0 {
		o.Weight = ext.Weight
	}

	db.Save(o)

	return nil
}
