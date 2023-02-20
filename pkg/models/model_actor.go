package models

import (
	"time"

	"github.com/avast/retry-go/v4"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	Name   string  `gorm:"unique_index" json:"name" xbvrbackup:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"-" xbvrbackup:"-"`
	Count  int     `json:"count" xbvrbackup:"-"`

	AvailCount int `json:"avail_count" xbvrbackup:"-"`
}

func (i *Actor) Save() error {
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

func (i *Actor) CountActorTags() {
	db, _ := GetDB()
	defer db.Close()

	type CountResults struct {
		ID            int
		Cnt           int
		Existingcnt   int
		IsAvailable   int
		Existingavail int
	}

	var results []CountResults

	db.Model(&Actor{}).
		Select("actors.id, count as existingcnt, count(*) cnt, sum(scenes.is_available ) is_available, avail_count as existingavail").
		Group("actors.id").
		Joins("join scene_cast on scene_cast.actor_id = actors.id").
		Joins("join scenes on scenes.id=scene_cast.scene_id and scenes.deleted_at is null").
		Scan(&results)

	for i := range results {
		var actor Actor
		if results[i].Cnt != results[i].Existingcnt || results[i].IsAvailable != results[i].Existingavail {
			db.First(&actor, results[i].ID)
			actor.Count = results[i].Cnt
			actor.AvailCount = results[i].IsAvailable
			actor.Save()
		}
	}
}
