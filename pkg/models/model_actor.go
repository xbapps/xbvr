package models

import (
	"time"

	"github.com/avast/retry-go/v3"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	Name   string  `gorm:"unique_index" json:"name" xbvrbackup:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"-" xbvrbackup:"-"`
	Count  int     `json:"count" xbvrbackup:"-"`
}

func (i *Actor) Save() error {
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
