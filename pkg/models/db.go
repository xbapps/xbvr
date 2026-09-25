package models

import (
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
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
var commonConnection *gorm.DB

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

// sqliteWriteDSN appends per-connection pragmas so concurrent scrapers ride
// out lock contention instead of dying with "database is locked" (#2139).
// The driver's default 5s busy_timeout is too short when dozens of scrapers
// pile up, and delete-mode readers block the writer wholesale; WAL removes
// the reader/writer conflict and 30s covers writer queues. mattn/go-sqlite3
// applies these to every pooled connection, unlike PRAGMAs issued after open.
func sqliteWriteDSN(dsn string) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "_busy_timeout=30000&_journal_mode=WAL&_synchronous=NORMAL"
}

func openDB() (*gorm.DB, error) {
	dsn := dbConn.DSN
	if dbConn.Driver == "sqlite3" {
		dsn = sqliteWriteDSN(dsn)
	}
	return gorm.Open(dbConn.Driver, dsn)
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
		retry.Attempts(30),
		retry.Delay(time.Second),
		retry.MaxDelay(15*time.Second),
		retry.DelayType(retry.BackOffDelay),
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
			db, err = openDB()
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

func GetCommonDB() (*gorm.DB, error) {
	if common.EnvConfig.DebugSQL {
		log.Debug("Getting Common DB handle from ", common.GetCallerFunctionName())
	}

	var err error

	if commonConnection != nil {
		return commonConnection, nil
	}
	err = retry.Do(
		func() error {
			commonConnection, err = openDB()
			commonConnection.LogMode(common.EnvConfig.DebugSQL)
			commonConnection.DB().SetConnMaxIdleTime(4 * time.Minute)
			if common.DBConnectionPoolSize > 0 {
				commonConnection.DB().SetMaxOpenConns(common.DBConnectionPoolSize)
			}
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to connect to database ", err)
	}

	return commonConnection, nil
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

	return err == nil
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
	GetCommonDB()
}
