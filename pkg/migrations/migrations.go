package migrations

import (
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/gormigrate.v1"
)

func Migrate() {
	db, _ := models.GetDB()

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "0001",
			Migrate: func(tx *gorm.DB) error {
				return tx.
					AutoMigrate(&models.Scene{}).
					AutoMigrate(&models.SceneCuepoint{}).
					AutoMigrate(&models.Actor{}).
					AutoMigrate(&models.Tag{}).
					AutoMigrate(&models.File{}).
					AutoMigrate(&models.Volume{}).
					AutoMigrate(&models.History{}).
					AutoMigrate(&models.Site{}).
					AutoMigrate(&models.KV{}).Error
			},
		},
		{
			ID: "0002",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					VideoAvgFrameRateVal float64
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0003",
			Migrate: func(tx *gorm.DB) error {
				var files []models.File
				tx.Model(&files).Find(&files)

				for i := range files {
					err := files[i].CalculateFramerate()
					if err == nil {
						files[i].Save()
					}
				}
				return nil
			},
		},
		{
			ID: "0004",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					NeedsUpdate bool
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
	})

	if err := m.Migrate(); err != nil {
		common.Log.Fatalf("Could not migrate: %v", err)
	}
	common.Log.Printf("Migration did run successfully")

	db.Close()
}
