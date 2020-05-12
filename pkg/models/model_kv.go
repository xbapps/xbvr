package models

import (
	"github.com/xbapps/xbvr/pkg/common"
)

type KV struct {
	Key   string `json:"key" gorm:"primary_key" gorm:"unique_index"`
	Value string `json:"value" sql:"type:text;"`
}

func (o *KV) Save() {
	db, _ := GetDB()
	db.Save(o)
	db.Close()
}

func (o *KV) Delete() {
	db, _ := GetDB()
	db.Delete(o)
	db.Close()
}

// Lock functions

func CreateLock(lock string) {
}

func CheckLock(lock string) bool {
	return false
}

func RemoveLock(lock string) {
}
