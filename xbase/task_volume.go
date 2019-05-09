package xbase

func RescanVolumes() {
	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Find(&vol)

	for i := range vol {
		vol[i].Rescan()
	}

	UpdateScenes()
}
