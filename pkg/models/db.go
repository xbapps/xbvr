package models

import (
	"path/filepath"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xbapps/xbvr/pkg/common"
)

var log = &common.Log

func GetDB() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", filepath.Join(common.AppDir, "main.db"))
	if err != nil {
		common.Log.Fatal("failed to connect database", err)
	}
	return db, nil
}

func init() {
	common.InitPaths()
}
