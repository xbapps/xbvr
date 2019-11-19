package models

import (
	"time"
)

type Site struct {
	ID         string    `gorm:"primary_key" json:"id"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url"`
	IsBuiltin  bool      `json:"is_builtin"`
	IsEnabled  bool      `json:"is_enabled"`
	LastUpdate time.Time `json:"last_update"`
}

func (i *Site) Save() error {
	db, _ := GetDB()
	err := db.Save(i).Error
	db.Close()
	return err
}

func (i *Site) GetIfExist(id string) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&Site{ID: id}).First(i).Error
}

func InitSites() {
	db, _ := GetDB()
	defer db.Close()

	scrapers := GetScrapers()
	for i := range scrapers {
		var st Site
		db.Where(&Site{ID: scrapers[i].ID}).FirstOrCreate(&st)
		st.Name = scrapers[i].Name
		st.AvatarURL = scrapers[i].AvatarURL
		st.IsBuiltin = true
		st.Save()
	}
}
