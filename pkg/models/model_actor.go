package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Actor struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name         string  `gorm:"unique_index" json:"name"`
	Scenes       []Scene `gorm:"many2many:scene_cast;" json:"-"`
	ActorID      string  `json:"_id"`
	Bio			 string  `json:"bio"`
	Birthday     string  `json:"birthday"`
	Ethnicity    string  `json:"ethnicity"`
	EyeColor     string  `json:"eye_color"`
	Facebook     string  `json:"facebook"`
	HairColor    string  `json:"hair_color"`
	Height       string  `json:"height"`
	HomepageURL  string  `json:"homepage_url"`
	ImageURL     string  `json:"image_url"`
	Instagram    string  `json:"instagram"`
	Measurements string  `json:"measurements"`
	Nationality  string  `json:"nationality"`
	Reddit       string  `json:"reddit"`
	Twitter      string  `json:"twitter"`
	Weight       string  `json:"weight"`
	Count        int     `json:"count"`
}

func (i *Actor) Save() error {
	db, _ := GetDB()
	err := db.Save(i).Error
	db.Close()
	return err
}

func ActorCreateUpdateFromExternal(db *gorm.DB, ext ScrapedActor) error {
	var o Actor
	db.Where(&Actor{Name: ext.Name}).First(&o)

	// TODO(jrebey): Figure out how to resolve aliases. Models like
	// https://www.hottiesvr.com/virtualreality/pornstar/id/640-Amy-Pink
	// have a bunch of aliases and don't get stats in the database.
	//
	// One idea is to check if there is an actor in the DB for the actual
	// actor. If not, we need to create it. Once the actual actor is in the
	// db, we can update the scene_cast table to point any aliases to the
	// actual actor. We can then delete the aliases from the db and update
	// the scene counts for the actor.

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

	if ext.Height != "" {
		o.Height = ext.Height
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

	if ext.Weight != "" {
		o.Weight = ext.Weight
	}

	db.Save(o)

	return nil
}
