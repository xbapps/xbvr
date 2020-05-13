package models

import (
	"fmt"
	"path/filepath"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/xbapps/xbvr/pkg/common"
)

var log = &common.Log

func GetDB() (*gorm.DB, error) {
	if common.DEBUG != "" {
		log.Debug("Getting DB handle from ", common.GetCallerFunctionName())
	}

	mysqlParams := fmt.Sprintf("%v:%v@(%v)/%v?charset=utf8&parseTime=True&loc=Local", common.MYSQL_USER, common.MYSQL_PASSWORD, common.MYSQL_HOST, common.MYSQL_DB)
	db, err := gorm.Open("mysql", mysqlParams)
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
