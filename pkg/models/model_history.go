package models

import (
	"time"

	"github.com/avast/retry-go/v4"
)

type History struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"created_at-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"updated_at"`

	SceneID   uint      `json:"scene_id" xbvrbackup:"-"`
	TimeStart time.Time `json:"time_start" xbvrbackup:"time_start"`
	TimeEnd   time.Time `json:"time_end" xbvrbackup:"time_end"`
	Duration  float64   `json:"duration" xbvrbackup:"duration"`
}

func (o *History) GetIfExist(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&History{ID: id}).First(o).Error
}

func (o *History) Save() {
	db, _ := GetDB()
	defer db.Close()

	var err error = retry.Do(
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
}

func (o *History) Delete() {
	db, _ := GetDB()
	db.Delete(&o)
	db.Close()
}
