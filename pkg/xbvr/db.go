package xbvr

import (
	"path/filepath"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func GetDB() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", filepath.Join(appDir, "main.db"))
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	return db, nil
}

func init() {
	initPaths()

	db, _ := GetDB()
	defer db.Close()

	db.AutoMigrate(&Scene{})
	db.AutoMigrate(&SceneCuepoint{})
	db.AutoMigrate(&Actor{})
	db.AutoMigrate(&Tag{})

	db.AutoMigrate(&File{})
	db.AutoMigrate(&Volume{})
	db.AutoMigrate(&History{})

	db.AutoMigrate(&Site{})
	db.AutoMigrate(&KV{})
}
