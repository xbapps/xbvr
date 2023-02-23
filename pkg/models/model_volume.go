package models

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

type Volume struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	Type           string    `json:"type" xbvrbackup:""`
	Path           string    `json:"path" xbvrbackup:""`
	Metadata       string    `json:"metadata" xbvrbackup:""`
	LastScan       time.Time `json:"last_scan" xbvrbackup:""`
	IsEnabled      bool      `json:"-" xbvrbackup:""`
	IsAvailable    bool      `json:"is_available" xbvrbackup:"-"`
	FileCount      int       `gorm:"-" json:"file_count" xbvrbackup:"-"`
	UnmatchedCount int       `gorm:"-" json:"unmatched_count" xbvrbackup:"-"`
	TotalSize      int64     `gorm:"-" json:"total_size" xbvrbackup:"-"`
}

func IsDirectoryEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.ReadDir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func (o *Volume) IsMounted() bool {
	switch o.Type {
	case "local":
		if _, err := os.Stat(o.Path); os.IsNotExist(err) {
			return false
		}
		if isEmpty, err := IsDirectoryEmpty(o.Path); isEmpty || err != nil {
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

	var files []File
	for i := range vol {
		isMounted := vol[i].IsMounted()
		if isMounted != vol[i].IsAvailable {
			vol[i].IsAvailable = vol[i].IsMounted()
			vol[i].Save()

			// update the status of any scene with a file on that volume
			db.
				Model(&files).
				Where("volume_id = ?", vol[i].ID).
				Find(&files)
			for _, file := range files {
				var scene Scene
				scene.GetIfExistByPK(file.SceneID)
				scene.UpdateStatus()
			}
		}
	}
}
