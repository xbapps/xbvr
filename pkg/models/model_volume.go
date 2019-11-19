package models

import (
	"context"
	"os"
	"time"

	"github.com/putdotio/go-putio/putio"
	"golang.org/x/oauth2"
)

type Volume struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Type           string    `json:"type"`
	Path           string    `json:"path"`
	Metadata       string    `json:"metadata"`
	LastScan       time.Time `json:"last_scan"`
	IsEnabled      bool      `json:"-"`
	IsAvailable    bool      `json:"is_available"`
	FileCount      int       `gorm:"-" json:"file_count"`
	UnmatchedCount int       `gorm:"-" json:"unmatched_count"`
	TotalSize      int64     `gorm:"-" json:"total_size"`
}

func (o *Volume) IsMounted() bool {
	switch o.Type {
	case "local":
		if _, err := os.Stat(o.Path); os.IsNotExist(err) {
			return false
		}
		return true
	case "putio":
		return true
	default:
		return false
	}
}

func (o *Volume) Save() error {
	db, _ := GetDB()
	err := db.Save(o).Error
	db.Close()
	return err
}

func (o *Volume) Files() []File {
	var allFiles []File
	db, _ := GetDB()
	db.Preload("Volume").Where("volume_id = ?", o.ID).Find(&allFiles)
	db.Close()
	return allFiles
}

func (o *Volume) GetPutIOClient() *putio.Client {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: o.Metadata})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	return putio.NewClient(oauthClient)
}

func CheckVolumes() {
	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Find(&vol)

	for i := range vol {
		vol[i].IsAvailable = vol[i].IsMounted()
		vol[i].Save()
	}
}
