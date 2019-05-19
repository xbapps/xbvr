package xbvr

import (
	"time"
)

type Image struct {
	ID        uint       `gorm:"primary_key" json:"-"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`

	ActorID     uint   `json:"-"`
	SceneID     uint   `json:"-"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Orientation string `json:"orientation"`
}

func (i *Image) GetPath() string {
	return ""
}

func (i *Image) Save() error {
	db, _ := GetDB()
	err := db.Save(i).Error
	db.Close()
	return err
}

func (i *Image) Delete() error {
	db, _ := GetDB()
	err := db.Delete(i).Error
	db.Close()

	// TODO: delete downloaded file

	return err
}
