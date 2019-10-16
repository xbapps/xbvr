package xbvr

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/gammazero/nexus/v3/client"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/vansante/go-ffprobe"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/cheggaaa/pb.v1"
)

func RescanVolumes() {
	if !models.CheckLock("rescan") {
		models.CreateLock("rescan")

		models.CheckVolumes()

		db, _ := models.GetDB()
		defer db.Close()

		tlog := log.WithFields(logrus.Fields{"task": "rescan"})

		tlog.Infof("Start scanning volumes")

		var vol []models.Volume
		db.Find(&vol)

		for i := range vol {
			log.Infof("Scanning %v", vol[i].Path)

			if vol[i].IsMounted() {
				allowedExt := []string{".mp4", ".avi", ".wmv", ".mpeg4", ".mov"}

				var procList []string
				_ = filepath.Walk(vol[i].Path, func(path string, f os.FileInfo, err error) error {
					if !f.Mode().IsDir() {
						// Make sure the filename should be considered
						if !strings.HasPrefix(filepath.Base(path), ".") && funk.Contains(allowedExt, strings.ToLower(filepath.Ext(path))) {
							var fl models.File
							err = db.Where(&models.File{Path: filepath.Dir(path), Filename: filepath.Base(path)}).First(&fl).Error

							if err == gorm.ErrRecordNotFound || fl.VolumeID == 0 || fl.VideoDuration == 0 || fl.VideoProjection == "" {
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

					var birthtime time.Time
					if fTimes.HasBirthTime() {
						birthtime = fTimes.BirthTime()
					} else {
						birthtime = fTimes.ModTime()
					}
					var fl models.File
					db.Where(&models.File{
						Path:     filepath.Dir(pth),
						Filename: filepath.Base(pth),
					}).FirstOrCreate(&fl)

					fl.Size = fStat.Size()
					fl.CreatedTime = birthtime
					fl.UpdatedTime = fTimes.ModTime()
					fl.VolumeID = vol[i].ID

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
						if dur, err := strconv.ParseFloat(vs.Duration, 64); err == nil {
							fl.VideoDuration = dur
						}

						if vs.Height*2 == vs.Width || vs.Width > vs.Height {
							fl.VideoProjection = "180_sbs"
						}

						if vs.Height == vs.Width {
							fl.VideoProjection = "360_tb"
						}
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
						db, _ := models.GetDB()
						db.Delete(&allFiles[i])
						db.Close()
					}
				}
			}
		}

		// Match Scene to File

		var files []models.File
		var scenes []models.Scene
		var changed = false

		tlog.Infof("Matching Scenes to known filenames")
		db.Model(&models.File{}).Find(&files)

		for i := range files {
			fn := files[i].Filename

			err := db.Raw("select scenes.* from scenes, json_each(scenes.filenames_arr) where lower(json_each.value) = ? group by scenes.scene_id", strings.ToLower(path.Base(fn))).Scan(&scenes).Error
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
		db.Model(&models.Scene{}).Find(&scenes)

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

				var newestFileDate time.Time
				for j := range files {
					if files[j].Exists() {
						if files[j].CreatedTime.Before(newestFileDate) || newestFileDate.IsZero() {
							newestFileDate = files[j].CreatedTime
						}
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

				if !newestFileDate.Equal(scenes[i].AddedDate) && !newestFileDate.IsZero() {
					scenes[i].AddedDate = newestFileDate
					changed = true
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

		tlog.Infof("Scanning complete")

		// Inform UI about state change
		publisher, err := client.ConnectNet(context.Background(), "ws://"+common.WsAddr+"/ws", client.Config{Realm: "default"})
		if err == nil {
			publisher.Publish("state.change.optionsFolders", nil, nil, nil)
			publisher.Close()
		}

	}

	models.RemoveLock("rescan")
}
