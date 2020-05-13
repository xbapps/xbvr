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

// Lock functions

func CreateLock(lock string) {
	obj := KV{Key: "lock-" + lock, Value: "1"}
	obj.Save()

	common.PublishWS("lock.change", map[string]interface{}{"name": lock, "locked": true})
}

func CheckLock(lock string) bool {
	db, _ := GetDB()
	defer db.Close()

	var obj KV
	err := db.Where(&KV{Key: "lock-" + lock}).First(&obj).Error
	if err == nil {
		return true
	}
	return false
}

func RemoveLock(lock string) {
	db, _ := GetDB()
	defer db.Close()

	db.Where("key = ?", "lock-"+lock).Delete(&KV{})

	common.PublishWS("lock.change", map[string]interface{}{"name": lock, "locked": false})
}

func init() {
	common.InitPaths()
}
