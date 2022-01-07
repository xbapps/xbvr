package models

import (
	"time"

	"github.com/avast/retry-go/v3"
)

// Playlist data model
type Playlist struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Name         string `json:"name"`
	Ordering     int    `json:"ordering"`
	IsSystem     bool   `json:"is_system"`
	IsDeoEnabled bool   `json:"is_deo_enabled"`
	IsSmart      bool   `json:"is_smart"`
	SearchParams string `json:"search_params" sql:"type:text;"`
}

func (o *Playlist) Save() error {
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
