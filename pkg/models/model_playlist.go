package models

import (
	"time"
)

// Playlist data model
type Playlist struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Name         string `json:"name"`
	IsDeoEnabled bool   `json:"is_deo_enabled"`
	IsSmart      bool   `json:"is_smart"`
	SearchParams string `json:"search_params" sql:"type:text;"`
}

func (o *Playlist) Save() error {
	db, _ := GetDB()
	err := db.Save(o).Error
	db.Close()
	return err
}
