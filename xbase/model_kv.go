package xbase

type KV struct {
	Key   string `json:"key" gorm:"primary_key" gorm:"unique_index"`
	Value string `json:"value"`
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
	obj := KV{Key: "lock-" + lock, Value: "1"}
	obj.Save()
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
}
