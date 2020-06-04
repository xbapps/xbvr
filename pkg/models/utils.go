package models

type DMSData struct {
	Sites        []string `json:"sites"`
	Actors       []string `json:"actors"`
	Tags         []string `json:"tags"`
	ReleaseGroup []string `json:"release_group"`
	Volumes      []Volume `json:"volumes"`
}

func GetDMSData() DMSData {
	db, _ := GetDB()
	defer db.Close()

	// Get all accessible scenes
	var scenes []Scene
	tx := db.
		Model(&scenes).
		Preload("Cast").
		Preload("Tags").
		Preload("Files")

	tx = tx.Where("is_accessible = ?", 1).Where("is_available = ?", 1)

	// Available sites
	tx.Group("site").Find(&scenes)
	var outSites []string
	for i := range scenes {
		if scenes[i].Site != "" {
			outSites = append(outSites, scenes[i].Site)
		}
	}

	// Available release dates (YYYY-MM)
	switch db.Dialect().GetName() {
	case "mysql":
		tx.Select("DATE_FORMAT(release_date, '%Y-%m') as release_date_text").
			Group("DATE_FORMAT(release_date, '%Y-%m')").Find(&scenes)
	case "sqlite3":
		tx.Select("strftime('%Y-%m', release_date) as release_date_text").
			Group("strftime('%Y-%m', release_date)").Find(&scenes)
	}
	var outRelease []string
	for i := range scenes {
		outRelease = append(outRelease, scenes[i].ReleaseDateText)
	}

	// Available tags
	tx.Joins("left join scene_tags on scene_tags.scene_id=scenes.id").
		Joins("left join tags on tags.id=scene_tags.tag_id").
		Group("tags.name").Order("tags.name asc").Select("tags.name as release_date_text").Find(&scenes)

	var outTags []string
	for i := range scenes {
		if scenes[i].ReleaseDateText != "" {
			outTags = append(outTags, scenes[i].ReleaseDateText)
		}
	}

	// Available actors
	tx.Joins("left join scene_cast on scene_cast.scene_id=scenes.id").
		Joins("left join actors on actors.id=scene_cast.actor_id").
		Group("actors.name").Order("actors.name asc").Select("actors.name as release_date_text").Find(&scenes)

	var outCast []string
	for i := range scenes {
		if scenes[i].ReleaseDateText != "" {
			outCast = append(outCast, scenes[i].ReleaseDateText)
		}
	}

	// Available volumes
	var vol []Volume
	db.Where("is_available = ?", true).Find(&vol)

	return DMSData{Sites: outSites, Tags: outTags, Actors: outCast, Volumes: vol, ReleaseGroup: outRelease}
}
