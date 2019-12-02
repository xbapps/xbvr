package models

import (
	"time"
)

type Actor struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`

	Name   string  `gorm:"unique_index" json:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"-"`
	Count  int     `json:"count"`
}

func (i *Actor) Save() error {
	db, _ := GetDB()
	err := db.Save(i).Error
	db.Close()
	return err
}
