package externalreference

import (
	"time"

	"github.com/xbapps/xbvr/pkg/models"
)

// check if the field was modified by the user, if so don't change it
func CheckAndSetStringActorField(actor_field *string, fieldName string, newValue string, actor_id uint) bool {
	if *actor_field == newValue {
		return false
	}
	if *actor_field == "" {
		*actor_field = newValue
		return true
	}

	// check if the field was modified by the user, if so don't change it
	db, _ := models.GetDB()
	defer db.Close()
	var action models.ActionActor
	db.Where("source = 'edit_actor' and actor_id = ? and changed_column = ?", actor_id, fieldName).Order("ID desc").First(&action)
	if action.NewValue != "" && action.NewValue != "0" {
		return false
	}

	*actor_field = newValue
	return true
}

// check if the field was modified by the user, if so don't change it
func CheckAndSetIntActorField(actor_field *int, fieldName string, newValue int, actor_id uint) bool {
	if *actor_field == newValue {
		return false
	}
	if *actor_field == 0 {
		*actor_field = newValue
		return true
	}

	// check if the field was modified by the user, if so don't change it
	db, _ := models.GetDB()
	defer db.Close()
	var action models.ActionActor
	db.Where("source = 'edit_actor' and actor_id = ? and changed_column = ?", actor_id, fieldName).Order("ID desc").First(&action)
	if action.NewValue != "" && action.NewValue != "0" {
		return false
	}

	*actor_field = newValue
	return true
}

// check if the field was modified by the user, if so don't change it
func CheckAndSetDateActorField(actor_field *time.Time, fieldName string, newValue string, actor_id uint) bool {
	bd, err := time.Parse("2006-01-02", newValue)
	if err != nil {
		return false
	}
	if bd.Equal(*actor_field) {
		return false
	}
	if actor_field.IsZero() {
		*actor_field = bd
		return true
	}

	// check if the field was modified by the user, if so don't change it
	db, _ := models.GetDB()
	defer db.Close()
	var action models.ActionActor
	db.Where("source = 'edit_actor' and actor_id = ? and changed_column = ?", actor_id, fieldName).Order("ID desc").First(&action)
	if action.NewValue != "" && action.NewValue != "0001-01-01T00:00:00Z" {
		return false
	}

	*actor_field = bd
	return true
}
