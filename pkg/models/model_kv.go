package models

import "github.com/avast/retry-go/v4"

type KV struct {
	Key   string `json:"key" gorm:"primary_key"`
	Value string `json:"value" sql:"type:text;"`
}

func (o *KV) Save() {
	db, _ := GetDB()
	defer db.Close()

	var err error = retry.Do(
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
