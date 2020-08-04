package models

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/avast/retry-go"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	Name   string  `gorm:"unique_index" json:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"-"`
	Count  int     `json:"count"`
}

func (i *Actor) Save() error {
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&i).Error
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

func ConvertName(searchString string, aliases [][]string) string {
	for _,v := range aliases {
		if stringInSlice(searchString, v) {
			log.Infof("Model alias found for: %v (%v)", searchString, v[0])
			return v[0]
		}
	}
	return searchString
}

func GetModelAliases() [][]string {
	jsonFile, err := os.Open("./pkg/models/model_aliases.json")
	if err != nil {
		log.Errorln(err)
		return nil
	}
	defer jsonFile.Close()

	var aliases [][]string
	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &aliases)

	log.Infoln("Found model aliases")

	return aliases
}

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}
