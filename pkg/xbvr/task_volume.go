package xbvr

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/vansante/go-ffprobe"
	"gopkg.in/cheggaaa/pb.v1"
)

func RescanVolumes() {
	if !CheckLock("rescan") {
		CreateLock("rescan")

		CheckVolumes()

		db, _ := GetDB()
		defer db.Close()

		tlog := log.WithFields(logrus.Fields{"task": "rescan"})

		tlog.Infof("Start scanning volumes")

		var vol []Volume
		db.Find(&vol)

		for i := range vol {
			log.Infof("Scanning %v", vol[i].Path)

			if vol[i].IsMounted() {
				notAllowedFn := []string{".DS_Store", ".tmp"}
				allowedExt := []string{".mp4", ".avi", ".wmv", ".mpeg4", ".mov"}

				var procList []string
				_ = filepath.Walk(vol[i].Path, func(path string, f os.FileInfo, err error) error {
					if !f.Mode().IsDir() {
						// Make sure the filename should be considered
						if !funk.Contains(notAllowedFn, filepath.Base(path)) && funk.Contains(allowedExt, strings.ToLower(filepath.Ext(path))) {
							var fl File
							err = db.Where(&File{Path: filepath.Dir(path), Filename: filepath.Base(path)}).First(&fl).Error

							if err == gorm.ErrRecordNotFound {
								procList = append(procList, path)
							}
						}
					}
					return nil
				})

				bar := pb.StartNew(len(procList))
				bar.Output = nil
				for j, pth := range procList {
					fStat, _ := os.Stat(pth)
					fTimes, _ := times.Stat(pth)

					var fl File
					fl = File{
						Path:        filepath.Dir(pth),
						Filename:    filepath.Base(pth),
						Size:        fStat.Size(),
						CreatedTime: fTimes.BirthTime(),
						UpdatedTime: fTimes.ModTime(),
					}

					ffdata, err := ffprobe.GetProbeData(pth, time.Second*3)
					if err != nil {
						tlog.Errorf("Error running ffprobe", pth, err)
					} else {
						vs := ffdata.GetFirstVideoStream()
						bitRate, _ := strconv.Atoi(vs.BitRate)
						fl.VideoAvgFrameRate = vs.AvgFrameRate
						fl.VideoBitRate = bitRate
						fl.VideoCodecName = vs.CodecName
						fl.VideoWidth = vs.Width
						fl.VideoHeight = vs.Height
					}

					err = fl.Save()
					if err != nil {
						tlog.Errorf("New file %s, but got error %s", pth, err)
					}

					bar.Increment()
					tlog.Infof("Scanning %v (%v/%v)", vol[i].Path, j+1, len(procList))
				}

				bar.Finish()

				vol[i].LastScan = time.Now()
				vol[i].Save()

				// Check if files are still present at the location
				allFiles := vol[i].Files()
				for i := range allFiles {
					if !allFiles[i].Exists() {
						log.Info(allFiles[i].GetPath())
						db, _ := GetDB()
						db.Delete(&allFiles[i])
						db.Close()
					}
				}
			}
		}

		// Match Scene to File

		var files []File
		var scenes []Scene
		var changed = false

		tlog.Infof("Matching Scenes to known filenames")
		db.Model(&File{}).Find(&files)

		for i := range files {
			fn := files[i].Filename

			err := db.Raw("select scenes.* from scenes, json_each(scenes.filenames_arr) where json_each.value = ? group by scenes.scene_id", path.Base(fn)).Scan(&scenes).Error
			if err != nil {
				log.Error(err, "when matching "+path.Base(fn))
			}

			if len(scenes) >= 1 {
				files[i].SceneID = scenes[0].ID
				files[i].Save()
			}

			if (i % 50) == 0 {
				tlog.Infof("Matching Scenes to known filenames (%v/%v)", i+1, len(files))
			}
		}

		// Update scene statuses

		tlog.Infof("Update status of Scenes")
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

			if (i % 70) == 0 {
				tlog.Infof("Update status of Scenes (%v/%v)", i+1, len(scenes))
			}
		}

		log.WithFields(logrus.Fields{"task": "rescan"}).Infof("Scanning complete")
	}

	RemoveLock("rescan")
}
