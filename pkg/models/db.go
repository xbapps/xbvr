package models

import (
	"path/filepath"

	"github.com/avast/retry-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xbapps/xbvr/pkg/common"
)

var log = &common.Log

type DBModel interface {
	Save() error
}

func SaveWithRetry(db *gorm.DB, i interface{}) error {
	var err error
	err = retry.Do(
		func() error {
			err = db.Save(i).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

func GetDB() (*gorm.DB, error) {
	if common.DEBUG != "" {
		log.Debug("Getting DB handle from ", common.GetCallerFunctionName())
	}

	var db *gorm.DB
	var err error

	err = retry.Do(
		func() error {
			db, err = gorm.Open("sqlite3", "file:"+filepath.Join(common.AppDir, "main.db")+"?"+common.SQLITE_PARAMS)
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to connect to database ", err)
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
