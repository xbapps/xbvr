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
	if common.DEBUG != "" {
		log.Debug("Getting DB handle from ", common.GetCallerFunctionName())
	}

	db, err := gorm.Open("sqlite3", "file:"+filepath.Join(common.AppDir, "main.db")+"?"+common.SQLITE_PARAMS)
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	return db, nil
}

func init() {
	common.InitPaths()
}
