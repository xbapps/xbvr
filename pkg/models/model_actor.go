package models

import (
	"time"

	"github.com/avast/retry-go/v3"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Name   string  `gorm:"unique_index" json:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"-"`
	Count  int     `json:"count"`
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
