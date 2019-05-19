package xbvr

import (
	"time"
)

type Actor struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	Name  string `gorm:"unique_index" json:"name"`
	Count int    `json:"count"`
}
