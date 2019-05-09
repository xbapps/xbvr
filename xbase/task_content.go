package xbase

import (
	"path"

	"github.com/go-test/deep"
)

func RenameTags() {
	db, _ := GetDB()
	defer db.Close()

	var scenes []Scene
	db.Find(&scenes)

	for i := range scenes {
		currentTags := make([]Tag, 0)
		db.Model(&scenes[i]).Related(&currentTags, "Tags")

		newTags := make([]Tag, 0)
		for j := range currentTags {
			nt := Tag{}
			if convertTag(currentTags[j].Name) != "" {
				db.Where(&Tag{Name: convertTag(currentTags[j].Name)}).FirstOrCreate(&nt)
				newTags = append(newTags, nt)
			}
		}

		diffs := deep.Equal(currentTags, newTags)
		if len(diffs) > 0 {
			for j := range currentTags {
				db.Model(&scenes[i]).Association("Tags").Delete(&currentTags[j])
			}

			for j := range newTags {
				db.Model(&scenes[i]).Association("Tags").Append(&newTags[j])
			}
		}

	}
}

func CountTags() {
	db, _ := GetDB()
	defer db.Close()

	var tags []Tag
	db.Model(&Tag{}).Find(&tags)

	for i := range tags {
		var scenes []Scene
		db.Model(tags[i]).Related(&scenes, "Scenes")

		tags[i].Count = len(scenes)
		tags[i].Save()
	}

	db.Where("count = ?", 0).Delete(&Tag{})
}

func UpdateScenes() {
	db, _ := GetDB()
	defer db.Close()

	var files []File
	var scenes []Scene
	var changed = false

	db.Model(&File{}).Find(&files)

	for i := range files {
		fn := files[i].Filename

		var pfn PossibleFilename
		db.Where("name LIKE ?", path.Base(fn)).First(&pfn)
		db.Model(&pfn).Preload("Cast").Preload("Tags").Related(&scenes, "Scenes")

		if len(scenes) == 1 {
			files[i].SceneID = scenes[0].ID
			files[i].Save()
		}
	}

	db.Model(&Scene{}).Find(&scenes)

	for i := range scenes {
		// Check if file with scene association exists
		files, err := scenes[i].GetFiles()
		if err != nil {
			return
		}

		changed = false

		if len(files) > 0 {
			if !scenes[i].IsAvailable {
				scenes[i].IsAvailable = true
				changed = true
			}
			for j := range files {
				if files[j].Exists() {
					if !scenes[i].IsAccessible {
						scenes[i].IsAccessible = true
						changed = true
					}
				} else {
					if scenes[i].IsAccessible {
						scenes[i].IsAccessible = false
						changed = true
					}
				}
			}
		} else {
			if scenes[i].IsAvailable {
				scenes[i].IsAvailable = false
				changed = true
			}
		}

		if changed {
			scenes[i].Save()
		}

	}

}
