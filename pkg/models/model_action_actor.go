package models

import (
	"time"

	"github.com/avast/retry-go/v4"
)

type ActionActor struct {
	ID        uint      `gorm:"primary_key" json:"id"  xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`

	ActorID       uint   `json:"actor_id" xbvrbackup:"-"`
	ActionType    string `json:"action_type" xbvrbackup:"action_type"`
	Source        string `json:"source" xbvrbackup:"source"`
	ChangedColumn string `json:"changed_column" xbvrbackup:"changed_column"`
	NewValue      string `json:"new_value" sql:"type:text;" xbvrbackup:"new_value"`
}

func (a *ActionActor) GetIfExist(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&ActionActor{ID: id}).First(a).Error
}

func (a *ActionActor) Save() {
	db, _ := GetDB()
	defer db.Close()

	var err error = retry.Do(
		func() error {
			err := db.Save(&a).Error
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

func AddActionActor(actorId uint, source string, actionType string, changedColumn string, newValue string) {
	action := ActionActor{
		ActorID:       actorId,
		Source:        source,
		ActionType:    actionType,
		ChangedColumn: changedColumn,
		NewValue:      newValue,
	}

	action.Save()
}

func Find(actorName string, actionType string, source string, changed_column string, newValue string) []ActionActor {
	db, _ := GetDB()
	defer db.Close()

	tx := db.Model(&ActionActor{})
	if actorName != "" {
		tx.Where("actor_name = ?", actorName)
	}
	if actionType != "" {
		tx.Where("action_type = ?", actionType)
	}
	if source != "" {
		tx.Where("source = ?", source)
	}
	if changed_column != "" {
		tx.Where("changed_column = ?", changed_column)
	}
	if newValue != "" {
		tx.Where("new_value = ?", newValue)
	}
	var results []ActionActor
	db.Find(&results)
	return results
}
