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

func GetModelAliases() (aliases ModelAliases, err error) {
	jsonFile, err := os.Open("./pkg/models/model_aliases.json")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &aliases)
	if err != nil {
		log.Debugln("There was an error getting model aliases:", err)
		return aliases, err
	}
	return aliases, nil
}
