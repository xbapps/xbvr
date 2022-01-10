package models

import "github.com/avast/retry-go/v3"

type KV struct {
	Key   string `json:"key" gorm:"primary_key" gorm:"unique_index"`
	Value string `json:"value" sql:"type:text;"`
}

func (o *KV) Save() {
	db, _ := GetDB()
	defer db.Close()

	var err error
	err = retry.Do(
		func() error {
			err := db.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *KV) Delete() {
	db, _ := GetDB()
	db.Delete(&o)
	db.Close()
}
