package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

var allowedVideoExt = []string{".mp4", ".avi", ".wmv", ".mpeg4", ".mov", ".mkv"}

func RescanVolumes(id int) {
	if !models.CheckLock("rescan") {
		models.CreateLock("rescan")
		defer models.RemoveLock("rescan")

		tlog := log.WithFields(logrus.Fields{"task": "rescan"})
		tlog.Infof("Start scanning volumes")

		models.CheckVolumes()

		db, _ := models.GetDB()
		defer db.Close()

		var vol []models.Volume
		if id > 0 {
			db.Where("id=?", id).Find(&vol)
		} else {
			db.Find(&vol)
		}

		for i := range vol {
			tlog.Infof("Scanning %v", vol[i].Path)

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
		var extrefs []models.ExternalReference

		tlog.Infof("Matching Scenes to known filenames")
		db.Model(&models.File{}).Where("files.scene_id = 0").Find(&files)

		escape := func(s string) string {
			var buffer bytes.Buffer
			json.HTMLEscape(&buffer, []byte(s))
			return buffer.String()
		}

		for i := range files {
			unescapedFilename := path.Base(files[i].Filename)
			filename := escape(unescapedFilename)
			filename2 := strings.Replace(filename, ".funscript", ".mp4", -1)
			filename3 := strings.Replace(filename, ".hsp", ".mp4", -1)
			filename4 := strings.Replace(filename, ".srt", ".mp4", -1)
			filename5 := strings.Replace(filename, ".cmscript", ".mp4", -1)
			err := db.Where("filenames_arr LIKE ? OR filenames_arr LIKE ? OR filenames_arr LIKE ? OR filenames_arr LIKE ? OR filenames_arr LIKE ?", `%"`+filename+`"%`, `%"`+filename2+`"%`, `%"`+filename3+`"%`, `%"`+filename4+`"%`, `%"`+filename5+`"%`).Find(&scenes).Error
			if err != nil {
				log.Error(err, " when matching "+unescapedFilename)
			}
			if len(scenes) == 0 && config.Config.Advanced.UseAltSrcInFileMatching {
				// check if the filename matches in external_reference record

				db.Preload("XbvrLinks").Where("external_source like 'alternate scene %' and external_data LIKE ? OR external_data LIKE ? OR external_data LIKE ? OR external_data LIKE ? OR external_data LIKE ?", `%"`+filename+`%`, `%"`+filename2+`%`, `%"`+filename3+`%`, `%"`+filename4+`%`, `%"`+filename5+`%`).Find(&extrefs)
				if len(extrefs) == 1 {
					if len(extrefs[0].XbvrLinks) == 1 {
						// the scene id will be the Internal DB Id from the associated link
						var scene models.Scene
						scene.GetIfExistByPK(extrefs[0].XbvrLinks[0].InternalDbId)
						// Add File to the list of Scene filenames
						var pfTxt []string
						err = json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt)
						if err != nil {
							continue
						}
						pfTxt = append(pfTxt, files[i].Filename)
						tmp, err := json.Marshal(pfTxt)
						if err == nil {
							scene.FilenamesArr = string(tmp)
						}
						scene.Save()
						scenes = append(scenes, scene)
					}
				}
			}
			if len(scenes) == 1 {
				files[i].SceneID = scenes[0].ID
				files[i].Save()
				scenes[0].UpdateStatus()
			} else {
				if config.Config.Storage.MatchOhash && config.Config.Advanced.StashApiKey != "" {
					hash := files[i].OsHash
					if len(hash) < 16 {
						// the has in xbvr is sometiomes < 16 pad with zeros
						paddingLength := 16 - len(hash)
						hash = strings.Repeat("0", paddingLength) + hash
					}
					queryVariable := `
				{"input":{
					"fingerprints": {					
						"value": "` + hash + `",
						"modifier": "INCLUDES"
					},				
					"page": 1
				}
				}`
					// call Stashdb graphql searching for os_hash
					stashMatches := scrape.GetScenePage(queryVariable)
					for _, match := range stashMatches.Data.QueryScenes.Scenes {
						if match.ID != "" {
							var externalRefLink models.ExternalReferenceLink
							db.Where(&models.ExternalReferenceLink{ExternalSource: "stashdb scene", ExternalId: match.ID}).First(&externalRefLink)
							if externalRefLink.ID != 0 {
								files[i].SceneID = externalRefLink.InternalDbId
								files[i].Save()
								var scene models.Scene
								scene.GetIfExistByPK(externalRefLink.InternalDbId)

								// add filename tyo the array
								var pfTxt []string
								json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt)
								pfTxt = append(pfTxt, files[i].Filename)
								tmp, _ := json.Marshal(pfTxt)
								scene.FilenamesArr = string(tmp)
								scene.Save()
								models.AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)

								scene.UpdateStatus()
								log.Infof("File %s matched to Scene %s matched using stashdb hash %s", path.Base(files[i].Filename), scene.SceneID, hash)
							}
						}
					}
				}
			}

			if (i % 50) == 0 {
				tlog.Infof("Matching Scenes to known filenames (%v/%v)", i+1, len(files))
			}
		}

		tlog.Infof("Generating heatmaps")

		GenerateHeatmaps(tlog)

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
}

func scanLocalVolume(vol models.Volume, db *gorm.DB, tlog *logrus.Entry) {
	if vol.IsMounted() {

		var videoProcList []string
		var scriptProcList []string
		var hspProcList []string
		var subtitlesProcList []string
		_ = filepath.Walk(vol.Path, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !f.Mode().IsDir() {
				// Make sure the filename should be considered
				if !strings.HasPrefix(filepath.Base(path), ".") && funk.Contains(allowedVideoExt, strings.ToLower(filepath.Ext(path))) {
					var fl models.File
					err = db.Where(&models.File{Path: filepath.Dir(path), Filename: filepath.Base(path)}).First(&fl).Error

					if err == gorm.ErrRecordNotFound || fl.VolumeID == 0 || fl.VideoDuration == 0 || fl.VideoProjection == "" || fl.Size != f.Size() || fl.OsHash == "" {
						videoProcList = append(videoProcList, path)
					}
				}

				if !strings.HasPrefix(filepath.Base(path), ".") && (filepath.Ext(path) == ".funscript" || strings.ToLower(filepath.Ext(path)) == ".cmscript") {
					scriptProcList = append(scriptProcList, path)
				}
				if !strings.HasPrefix(filepath.Base(path), ".") && filepath.Ext(path) == ".hsp" {
					hspProcList = append(hspProcList, path)
				}
				if !strings.HasPrefix(filepath.Base(path), ".") && (filepath.Ext(path) == ".srt" || filepath.Ext(path) == ".ssa" || filepath.Ext(path) == ".ass") {
					subtitlesProcList = append(subtitlesProcList, path)
				}
			}
			return nil
		})

		filenameSeparator := regexp.MustCompile("[ _.-]+")

		for j, path := range videoProcList {
			fStat, _ := os.Stat(path)
			fTimes, err := times.Stat(path)
			if err != nil {
				tlog.Errorf("Can't get the modification/creation times for %s, error: %s", path, err)
			}

			var birthtime time.Time
			if fTimes.HasBirthTime() {
				birthtime = fTimes.BirthTime()
			} else {
				birthtime = fTimes.ModTime()
			}
			var fl models.File
			db.Where(&models.File{
				Path:     filepath.Dir(path),
				Filename: filepath.Base(path),
				Type:     "video",
			}).FirstOrCreate(&fl)

			fl.Size = fStat.Size()
			fl.CreatedTime = birthtime
			fl.UpdatedTime = fTimes.ModTime()
			fl.VolumeID = vol.ID

			hash, err := Hash(path)
			if err == nil {
				fl.OsHash = fmt.Sprintf("%x", hash)
			}

			ffdata, err := ffprobe.GetProbeData(path, time.Second*5)
			if err != nil {
				tlog.Error("Error running ffprobe", path, err)
			} else {
				vs := ffdata.GetFirstVideoStream()
				if vs == nil {
					tlog.Error("No video stream in file ", path)
				} else {
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
					} else if ffdata.Format.DurationSeconds > 0.0 {
						fl.VideoDuration = ffdata.Format.DurationSeconds
					}
					fl.HasAlpha = false

					if vs.Height*2 == vs.Width || vs.Width > vs.Height {
						fl.VideoProjection = "180_sbs"
						nameparts := filenameSeparator.Split(strings.ToLower(filepath.Base(path)), -1)
						for i, part := range nameparts {
							if part == "mkx200" || part == "mkx220" || part == "rf52" || part == "fisheye190" || part == "vrca220" || part == "flat" {
								fl.VideoProjection = part
								break
							} else if part == "fisheye" || part == "f180" || part == "180f" {
								fl.VideoProjection = "fisheye"
								break
							} else if i < len(nameparts)-1 && (part+"_"+nameparts[i+1] == "mono_360" || part+"_"+nameparts[i+1] == "mono_180") {
								fl.VideoProjection = nameparts[i+1] + "_mono"
								break
							} else if i < len(nameparts)-1 && (part+"_"+nameparts[i+1] == "360_mono" || part+"_"+nameparts[i+1] == "180_mono") {
								fl.VideoProjection = part + "_mono"
								break
							}
						}
						if fl.VideoProjection == "mkx200" || fl.VideoProjection == "mkx220" || fl.VideoProjection == "rf52" || fl.VideoProjection == "fisheye190" || fl.VideoProjection == "vrca220" {
							// alpha passthrough only works with fisheye projections
							for _, part := range nameparts {
								if part == "alpha" {
									fl.HasAlpha = true
									break
								}
							}
						}
					}

					if vs.Height == vs.Width {
						fl.VideoProjection = "360_tb"
					}

					fl.CalculateFramerate()
				}
			}

			err = fl.Save()
			if err != nil {
				tlog.Errorf("New file %s, but got error %s", path, err)
			}

			tlog.Infof("Scanning %v (%v/%v)", vol.Path, j+1, len(videoProcList))
		}

		for _, path := range scriptProcList {
			var fl models.File
			db.Where(&models.File{
				Path:     filepath.Dir(path),
				Filename: filepath.Base(path),
				Type:     "script",
			}).FirstOrCreate(&fl)

			fStat, _ := os.Stat(path)
			fTimes, _ := times.Stat(path)

			if fStat.Size() != fl.Size {
				fl.Size = fStat.Size()
				fl.HasHeatmap = false
				fl.VideoDuration = 0.0
			}

			if fl.VideoDuration < 0.01 {
				duration, err := getFunscriptDuration(path)
				if err == nil {
					fl.VideoDuration = duration
				}
			}

			fl.CreatedTime = fTimes.ModTime()
			fl.UpdatedTime = fTimes.ModTime()
			fl.VolumeID = vol.ID
			fl.Save()
		}

		for _, path := range hspProcList {
			ScanLocalHspFile(path, vol.ID, 0)
		}

		for _, path := range subtitlesProcList {
			ScanLocalSubtitlesFile(path, vol.ID, 0)
		}

		vol.LastScan = time.Now()
		vol.Save()

		var scene models.Scene
		// Check if files are still present at the location
		allFiles := vol.Files()
		for i := range allFiles {
			if !allFiles[i].Exists() {
				log.Info(allFiles[i].GetPath())
				db.Delete(&allFiles[i])
				if allFiles[i].SceneID != 0 {
					scene.GetIfExistByPK(allFiles[i].SceneID)
					scene.UpdateStatus()
				}
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
		if !files[i].IsDir() && funk.Contains(allowedVideoExt, strings.ToLower(filepath.Ext(files[i].Name))) {
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
				fl.Type = "video"
				fl.CreatedTime = files[i].CreatedAt.Time
				fl.UpdatedTime = files[i].UpdatedAt.Time
				fl.VolumeID = vol.ID
				fl.OsHash = files[i].OpensubtitlesHash
				fl.Save()
			}

			currentFileID = append(currentFileID, strconv.FormatInt(files[i].ID, 10))
		}
	}

	var scene models.Scene
	// Check if local files are present in listing
	allFiles := vol.Files()
	for i := range allFiles {
		if !funk.ContainsString(currentFileID, allFiles[i].Path) {
			log.Info(allFiles[i].GetPath())
			db.Delete(&allFiles[i])
			if allFiles[i].SceneID != 0 {
				scene.GetIfExistByPK(allFiles[i].SceneID)
				scene.UpdateStatus()
			}
		}
	}

	// Update volume info
	vol.IsAvailable = true
	vol.Path = "Put.io (" + acct.Username + ")"
	vol.LastScan = time.Now()
	vol.Save()
}
func RefreshSceneStatuses() {
	// refreshes the status of all scenes
	tlog := log.WithFields(logrus.Fields{"task": "rescan"})
	tlog.Infof("Update status of Scenes")
	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Model(&models.Scene{}).Find(&scenes)

	for i := range scenes {
		scenes[i].UpdateStatus()
		if (i % 70) == 0 {
			tlog.Infof("Update status of Scenes (%v/%v)", i+1, len(scenes))
		}
	}

	tlog.Infof("Scene status refresh complete")
}
func ScanLocalHspFile(path string, volID uint, sceneId uint) {
	db, _ := models.GetDB()
	defer db.Close()

	var fl models.File
	db.Where(&models.File{
		Path:     filepath.Dir(path),
		Filename: filepath.Base(path),
		Type:     "hsp",
	}).FirstOrCreate(&fl)

	fStat, _ := os.Stat(path)
	fTimes, _ := times.Stat(path)

	fl.Size = fStat.Size()
	fl.CreatedTime = fTimes.ModTime()
	fl.UpdatedTime = fTimes.ModTime()
	fl.VolumeID = volID
	if sceneId > 0 {
		fl.SceneID = sceneId
	}
	fl.Save()

}

func ScanLocalSubtitlesFile(path string, volID uint, sceneId uint) {
	db, _ := models.GetDB()
	defer db.Close()

	var fl models.File
	db.Where(&models.File{
		Path:     filepath.Dir(path),
		Filename: filepath.Base(path),
		Type:     "subtitles",
	}).FirstOrCreate(&fl)

	fStat, _ := os.Stat(path)
	fTimes, _ := times.Stat(path)

	fl.Size = fStat.Size()
	fl.CreatedTime = fTimes.ModTime()
	fl.UpdatedTime = fTimes.ModTime()
	fl.VolumeID = volID
	if sceneId > 0 {
		fl.SceneID = sceneId
	}
	fl.Save()
}
