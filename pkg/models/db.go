package models

import (
	"path/filepath"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xbapps/xbvr/pkg/common"
)

func GetDB() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", filepath.Join(common.AppDir, "main.db"))
	if err != nil {
		common.Log.Fatal("failed to connect database", err)
	}
	return db, nil
}

func init() {
	common.InitPaths()

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
