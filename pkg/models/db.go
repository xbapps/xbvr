package models

import (
	"fmt"
	"path/filepath"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xbapps/xbvr/pkg/common"
)

var log = &common.Log

func GetDB() (*gorm.DB, error) {
	if common.DEBUG != "" {
		log.Debug("Getting DB handle from ", common.GetCallerFunctionName())
	}

	var dialect, params string
	switch common.DB_TYPE {
	case "mysql", "mariadb":
		dialect = "mysql"
		params = fmt.Sprintf("%v:%v@(%v)/%v?charset=utf8mb4&parseTime=True&loc=Local", common.DB_USER, common.DB_PASSWORD, common.DB_HOST, common.DB_NAME)
	case "sqlite3":
		dialect = "sqlite3"
		params = "file:" + filepath.Join(common.AppDir, "main.db") + "?" + common.SQLITE_PARAMS
	default:
		log.Fatal("Unknown database type: ", common.DB_TYPE)
	}
	db, err := gorm.Open(dialect, params)

	if common.SQL_DEBUG {
		db.LogMode(true)
	}

	if err != nil {
		log.Fatal("Failed to connect database: ", err)
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

	var obj KV
	db.Where(&KV{Key: "lock-" + lock}).Delete(&obj)

	common.PublishWS("lock.change", map[string]interface{}{"name": lock, "locked": false})
}

func init() {
	common.InitPaths()
}
