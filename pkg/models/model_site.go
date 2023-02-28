package models

import (
	"time"

	"github.com/avast/retry-go/v4"
)

type Site struct {
	ID         string    `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	Name       string    `json:"name"  xbvrbackup:"name"`
	AvatarURL  string    `json:"avatar_url" xbvrbackup:"-"`
	IsBuiltin  bool      `json:"is_builtin" xbvrbackup:"-"`
	IsEnabled  bool      `json:"is_enabled" xbvrbackup:"is_enabled"`
	LastUpdate time.Time `json:"last_update" xbvrbackup:"-"`
	Subscribed bool      `json:"subscribed" xbvrbackup:"subscribed"`
}

func (i *Site) Save() error {
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
