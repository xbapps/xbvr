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

func init() {
	common.InitPaths()
}
