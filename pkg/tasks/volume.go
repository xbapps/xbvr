package tasks

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/cheggaaa/pb.v1"
)

var allowedExt = []string{".mp4", ".avi", ".wmv", ".mpeg4", ".mov", ".mkv"}

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

			switch vol[i].Type {
			case "local":
				scanLocalVolume(vol[i], db, tlog)
			case "putio":
				scanPutIO(vol[i], db, tlog)
			}
		}

		// Match Scene to File
		var files []models.File
		var scenes []models.Scene

		tlog.Infof("Matching Scenes to known filenames")
		db.Model(&models.File{}).Where("files.scene_id = 0").Find(&files)

		for i := range files {
			fn := files[i].Filename
			err := db.Where("filenames_arr LIKE ?", fmt.Sprintf("%%\"%v\"%%", path.Base(fn))).Find(&scenes).Error
			if err != nil {
				log.Error(err, " when matching "+path.Base(fn))
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
			scenes[i].UpdateStatus()
			if (i % 70) == 0 {
				tlog.Infof("Update status of Scenes (%v/%v)", i+1, len(scenes))
			}
		}

		tlog.Infof("Scanning complete")

		// Inform UI about state change
		common.PublishWS("state.change.optionsStorage", nil)

		// Grab metrics
		var localFilesCount int64
		db.Model(models.File{}).
			Joins("left join volumes on files.volume_id = volumes.id").
			Where("volumes.type = ?", "local").
			Count(&localFilesCount)
		common.AddMetricPoint("local_files_count", float64(localFilesCount))

		var localFiles []models.File
		var localFilesSize int64 = 0
		db.Model(models.File{}).
			Joins("left join volumes on files.volume_id = volumes.id").
			Where("volumes.type = ?", "local").
			Scan(&localFiles)
		for _, v := range localFiles {
			localFilesSize = localFilesSize + v.Size
		}
		common.AddMetricPoint("local_files_size", float64(localFilesSize))

		r := models.RequestSceneList{}
		common.AddMetricPoint("scenes_scraped", float64(models.QueryScenes(r, false).Results))

		r = models.RequestSceneList{IsAvailable: optional.NewBool(true)}
		common.AddMetricPoint("scenes_downloaded", float64(models.QueryScenes(r, false).Results))

		r = models.RequestSceneList{IsWatched: optional.NewBool(true)}
		common.AddMetricPoint("scenes_watched_overall", float64(models.QueryScenes(r, false).Results))

		r = models.RequestSceneList{IsWatched: optional.NewBool(false), IsAvailable: optional.NewBool(true)}
		common.AddMetricPoint("scenes_downloaded_unwatched", float64(models.QueryScenes(r, false).Results))
	}

	models.RemoveLock("rescan")
}

func scanLocalVolume(vol models.Volume, db *gorm.DB, tlog *logrus.Entry) {
	if vol.IsMounted() {

		var procList []string
		_ = filepath.Walk(vol.Path, func(path string, f os.FileInfo, err error) error {
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
			fl.VolumeID = vol.ID

			ffdata, err := ffprobe.GetProbeData(pth, time.Second*3)
			if err != nil {
				tlog.Errorf("Error running ffprobe", pth, err)
			} else {
				vs := ffdata.GetFirstVideoStream()
				if vs.BitRate != "" {
					bitRate, _ := strconv.Atoi(vs.BitRate)
					fl.VideoBitRate = bitRate
				}
				fl.VideoAvgFrameRate = vs.AvgFrameRate
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

				fl.CalculateFramerate()
			}

			err = fl.Save()
			if err != nil {
				tlog.Errorf("New file %s, but got error %s", pth, err)
			}

			bar.Increment()
			tlog.Infof("Scanning %v (%v/%v)", vol.Path, j+1, len(procList))
		}

		bar.Finish()

		vol.LastScan = time.Now()
		vol.Save()

		// Check if files are still present at the location
		allFiles := vol.Files()
		for i := range allFiles {
			if !allFiles[i].Exists() {
				log.Info(allFiles[i].GetPath())
				db.Delete(&allFiles[i])
			}
		}
	}
}

func scanPutIO(vol models.Volume, db *gorm.DB, tlog *logrus.Entry) {
	client := vol.GetPutIOClient()

	acct, err := client.Account.Info(context.Background())
	if err != nil {
		vol.IsAvailable = false
		vol.Save()
		return
	}

	files, _, err := client.Files.List(context.Background(), -1)
	if err != nil {
		return
	}

	// Walk
	var currentFileID []string
	for i := range files {
		if !files[i].IsDir() && funk.Contains(allowedExt, strings.ToLower(filepath.Ext(files[i].Name))) {
			var fl models.File
			err = db.Where(&models.File{Path: strconv.FormatInt(files[i].ID, 10), Filename: files[i].Name}).First(&fl).Error

			if err == gorm.ErrRecordNotFound {
				var fl models.File
				db.Where(&models.File{
					Path:     strconv.FormatInt(files[i].ID, 10),
					Filename: files[i].Name,
				}).FirstOrCreate(&fl)
				fl.VideoProjection = "180_sbs"
				fl.Size = files[i].Size
				fl.CreatedTime = files[i].CreatedAt.Time
				fl.UpdatedTime = files[i].UpdatedAt.Time
				fl.VolumeID = vol.ID
				fl.Save()
			}

			currentFileID = append(currentFileID, strconv.FormatInt(files[i].ID, 10))
		}
	}

	// Check if local files are present in listing
	allFiles := vol.Files()
	for i := range allFiles {
		if !funk.ContainsString(currentFileID, allFiles[i].Path) {
			log.Info(allFiles[i].GetPath())
			db.Delete(&allFiles[i])
		}
	}

	// Update volume info
	vol.IsAvailable = true
	vol.Path = "Put.io (" + acct.Username + ")"
	vol.LastScan = time.Now()
	vol.Save()
}
