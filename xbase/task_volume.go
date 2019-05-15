package xbase

import (
	"path"

	"github.com/sirupsen/logrus"
)

func RescanVolumes() {
	if !CheckLock("rescan") {
		CreateLock("rescan")

		db, _ := GetDB()
		defer db.Close()

		log.WithFields(logrus.Fields{"task": "rescan"}).Infof("Start scanning volumes")

		var vol []Volume
		db.Find(&vol)

		for i := range vol {
			log.Infof("Scanning %v", vol[i].Path)
			vol[i].Rescan()
		}

		// Match Scene to File

		log.WithFields(logrus.Fields{"task": "rescan"}).Infof("Matching Scenes to known filenames")

		var files []File
		var scenes []Scene
		var changed= false

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

		// Update scene statuses

		log.WithFields(logrus.Fields{"task": "rescan"}).Infof("Update status of Scenes")

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

		log.WithFields(logrus.Fields{"task": "rescan"}).Infof("Scanning complete")
	}

	RemoveLock("rescan")
}
