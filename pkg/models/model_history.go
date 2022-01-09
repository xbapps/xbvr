package models

import (
	"time"

	"github.com/avast/retry-go/v3"
)

type History struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	SceneID   uint      `json:"scene_id"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
	Duration  float64   `json:"duration"`
}

func (o *History) GetIfExist(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&History{ID: id}).First(o).Error
}

func (o *History) Save() {
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
}

func (o *History) Delete() {
	db, _ := GetDB()
	db.Delete(&o)
	db.Close()
}
