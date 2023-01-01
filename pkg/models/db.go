package models

import (
	"strings"

	"github.com/avast/retry-go/v3"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xo/dburl"
)

var log = &common.Log
var dbConn *dburl.URL
var supportedDB = []string{"mysql", "sqlite3"}

func parseDBConnString() {
	var err error
	dbConn, err = dburl.Parse(common.DATABASE_URL)
	if err != nil {
		log.Fatal("Error parsing database connection ", common.DATABASE_URL, err)
	}
	_, ok := gorm.GetDialect(dbConn.Driver)
	if !ok || !funk.Contains(supportedDB, dbConn.Driver) {
		log.Fatal("Unsupported database: ", dbConn.Short())
	}
}

func GetDBConn() *dburl.URL {
	return dbConn
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
	if common.EnvConfig.DebugSQL {
		log.Debug("Getting DB handle from ", common.GetCallerFunctionName())
	}

	var db *gorm.DB
	var err error

	err = retry.Do(
		func() error {
			db, err = gorm.Open(dbConn.Driver, dbConn.DSN)
			db.LogMode(common.EnvConfig.DebugSQL)
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

	var obj KV
	db.Where(&KV{Key: "lock-" + lock}).Delete(&obj)

	common.PublishWS("lock.change", map[string]interface{}{"name": lock, "locked": false})
}

func RemoveAllLocks() {
	db, _ := GetDB()
	defer db.Close()

	var locks []KV
	err := db.Where("`key` like 'lock-%'").Find(&locks).Error
	if err != nil {
		return
	}

	for _, lock := range locks {
		lockName := strings.Replace(lock.Key, "lock-", "", 1)
		RemoveLock(lockName)
	}
}

func init() {
	common.InitPaths()
	common.InitLogging()
	parseDBConnString()
}
