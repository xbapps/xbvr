package migrations

import (
	"time"

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
		{
			ID: "0005",
			Migrate: func(tx *gorm.DB) error {
				type Volume struct {
					Type     string
					Metadata string
				}
				return tx.AutoMigrate(Volume{}).Error
			},
		},
		{
			ID: "0006",
			Migrate: func(tx *gorm.DB) error {
				var volumes []models.Volume
				tx.Model(&volumes).Find(&volumes)

				for i := range volumes {
					if volumes[i].Type == "" {
						volumes[i].Type = "local"
						volumes[i].Save()
					}
				}
				return nil
			},
		},
		{
			ID: "0007",
			Migrate: func(tx *gorm.DB) error {
				type Site struct {
					AvatarURL string
				}
				return tx.AutoMigrate(Site{}).Error
			},
		},
		{
			ID: "0010",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct{}
				type Actor struct {
					ID        uint
					CreatedAt time.Time
					UpdatedAt time.Time

					Name         string
					Scenes       []Scene
					ActorID      string
					Aliases      *string `sql:"type:text;"`
					Bio          string
					Birthday     string
					Ethnicity    string
					EyeColor     string
					Facebook     string
					HairColor    string
					Height       int
					HomepageURL  string
					ImageURL     string
					Instagram    string
					Measurements string
					Nationality  string
					Reddit       string
					Twitter      string
					Weight       int
					Count        int
				}
				return tx.AutoMigrate(Actor{}).Error
			},
		},
	})

	if err := m.Migrate(); err != nil {
		common.Log.Fatalf("Could not migrate: %v", err)
	}
	common.Log.Printf("Migration did run successfully")

	db.Close()
}
