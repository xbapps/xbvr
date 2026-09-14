package tasks

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func TestFindActorForEditReplay(t *testing.T) {
	db, err := models.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.AutoMigrate(&models.Actor{}).Error; err != nil {
		t.Fatal(err)
	}

	originalSetting := config.Config.Advanced.RestoreMissingActors
	if !originalSetting {
		t.Fatal("expected missing actor restoration to default to enabled")
	}
	defer func() {
		config.Config.Advanced.RestoreMissingActors = originalSetting
	}()

	t.Run("restores missing actors when enabled", func(t *testing.T) {
		const name = "Hatsumi Saki enabled"
		db.Unscoped().Where("name = ?", name).Delete(&models.Actor{})
		config.Config.Advanced.RestoreMissingActors = true

		actor, found := findActorForEditReplay(db, name)
		if !found || actor.ID == 0 {
			t.Fatal("expected missing actor to be restored")
		}
	})

	t.Run("skips missing actors when disabled", func(t *testing.T) {
		const name = "Hatsumi Saki disabled"
		db.Unscoped().Where("name = ?", name).Delete(&models.Actor{})
		config.Config.Advanced.RestoreMissingActors = false

		if _, found := findActorForEditReplay(db, name); found {
			t.Fatal("expected missing actor to remain deleted")
		}

		var count int
		db.Model(&models.Actor{}).Where("name = ?", name).Count(&count)
		if count != 0 {
			t.Fatalf("expected no restored actor, found %d", count)
		}
	})

	t.Run("finds existing localized actors when disabled", func(t *testing.T) {
		const name = "初美沙希"
		db.Unscoped().Where("name = ?", name).Delete(&models.Actor{})
		existing := models.Actor{Name: name}
		if err := db.Create(&existing).Error; err != nil {
			t.Fatal(err)
		}
		config.Config.Advanced.RestoreMissingActors = false

		actor, found := findActorForEditReplay(db, name)
		if !found || actor.ID != existing.ID {
			t.Fatal("expected existing localized actor to be found")
		}
	})
}
