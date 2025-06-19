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

// Default video extensions, used only when config is not available
var defaultVideoExt = []string{".mp4", ".avi", ".wmv", ".mpeg4", ".mov", ".mkv", ".m4v"}

func getVideoExtensions() []string {
	if config.Config.Storage.VideoExt != nil && len(config.Config.Storage.VideoExt) > 0 {
		return config.Config.Storage.VideoExt
	}
	return defaultVideoExt
}

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

		tlog.Infof("Matching files to known filenames")
		db.Model(&models.File{}).Where("files.scene_id = 0").Find(&files)

		escape := func(s string) string {
			var buffer bytes.Buffer
			json.HTMLEscape(&buffer, []byte(s))
			return buffer.String()
		}

		// Helper function to get base filename without extension
		getBaseFilename := func(filename string) string {
			return strings.TrimSuffix(path.Base(filename), filepath.Ext(filename))
		}

		// Helper function to match file to scene using FilenamesArr
		matchFileToScene := func(file *models.File, tlog *logrus.Entry) bool {
			baseFilename := getBaseFilename(file.Filename)
			// Search for any filename in FilenamesArr that has the same base name (ignoring extension)
			err := db.Where("filenames_arr LIKE ?", `%"`+escape(baseFilename)+`.%`).Find(&scenes).Error
			if err != nil {
				log.Error(err, " when matching "+file.Filename)
				return false
			}

			// If no direct match and alt source matching is enabled, try that
			if len(scenes) == 0 && config.Config.Advanced.UseAltSrcInFileMatching {
				db.Preload("XbvrLinks").Where("external_source like 'alternate scene %' and external_data LIKE ?", `%"`+escape(baseFilename)+`.%`).Find(&extrefs)
				if len(extrefs) == 1 && len(extrefs[0].XbvrLinks) == 1 {
					var scene models.Scene
					scene.GetIfExistByPK(extrefs[0].XbvrLinks[0].InternalDbId)
					
					// Add File to the list of Scene filenames
					var pfTxt []string
					err = json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt)
					if err == nil {
						pfTxt = append(pfTxt, file.Filename)
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
				file.SceneID = scenes[0].ID
				file.Save()
				scenes[0].UpdateStatus()
				tlog.Infof("Matched file %v to scene %v using filename", file.Filename, scenes[0].SceneID)
				return true
			}

			return false
		}

		for i, file := range files {
			// Try to match using FilenamesArr first
			if matchFileToScene(&file, tlog) {
				continue
			}

			// Try to find any file with the same base name that has a scene_id
			if file.Type == "video" || file.Type == "script" || file.Type == "hsp" || strings.HasPrefix(file.Type, "subtitle") {
				baseFilename := getBaseFilename(file.Filename)
				var matchingFile models.File
				
				// Look for any file with same base name that has a scene_id
				err := db.Where("scene_id != 0 AND filename LIKE ?", 
					baseFilename+".%").
					First(&matchingFile).Error

				if err == nil && matchingFile.SceneID != 0 {
					file.SceneID = matchingFile.SceneID

					// Add filename to scene's FilenamesArr
					var scene models.Scene
					scene.GetIfExistByPK(matchingFile.SceneID)
					
					var pfTxt []string
					if err := json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt); err == nil {
						pfTxt = append(pfTxt, file.Filename)
						if tmp, err := json.Marshal(pfTxt); err == nil {
							scene.FilenamesArr = string(tmp)
							scene.Save()
							scene.UpdateStatus()
						}
					}
					
					tlog.Infof("Auto-matched %v to scene %v via matching %v", file.Filename, scene.SceneID, matchingFile.Filename)
					continue
				}
			}

			// If still no match and we have StashDB integration, try that
			if file.Type == "video" && config.Config.Storage.MatchOhash && config.Config.Advanced.StashApiKey != "" {
				hash := file.OsHash
				if len(hash) < 16 {
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
				stashMatches := scrape.GetScenePage(queryVariable)
				for _, match := range stashMatches.Data.QueryScenes.Scenes {
					if match.ID != "" {
						var externalRefLink models.ExternalReferenceLink
						db.Where(&models.ExternalReferenceLink{ExternalSource: "stashdb scene", ExternalId: match.ID}).First(&externalRefLink)
						if externalRefLink.ID != 0 {
							file.SceneID = externalRefLink.InternalDbId
							file.Save()
							var scene models.Scene
							scene.GetIfExistByPK(externalRefLink.InternalDbId)

							var pfTxt []string
							json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt)
							pfTxt = append(pfTxt, file.Filename)
							tmp, _ := json.Marshal(pfTxt)
							scene.FilenamesArr = string(tmp)
							scene.Save()
							models.AddAction(scene.SceneID, "match", "filenames_arr", scene.FilenamesArr)

							scene.UpdateStatus()
							tlog.Infof("File %s matched to Scene %s using stashdb hash %s", path.Base(file.Filename), scene.SceneID, hash)
						}
					}
				}
			}

			if (i % 50) == 0 {
				tlog.Infof("Matching files to known filenames (%v/%v)", i+1, len(files))
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
				if !strings.HasPrefix(filepath.Base(path), ".") && funk.Contains(getVideoExtensions(), strings.ToLower(filepath.Ext(path))) {
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

			// Auto-match funscript with video if names match
			if fl.SceneID == 0 {
				// Get base filename without extension
				baseName := strings.TrimSuffix(fl.Filename, filepath.Ext(fl.Filename))
				
				// Look for matching video file
				var matchingVideo models.File
				err := db.Where("type = ? AND scene_id != 0 AND filename LIKE ?", 
					"video", 
					baseName+".%").
					First(&matchingVideo).Error

				if err == nil && matchingVideo.SceneID != 0 {
					fl.SceneID = matchingVideo.SceneID
					
					// Add funscript to scene's FilenamesArr
					var scene models.Scene
					scene.GetIfExistByPK(matchingVideo.SceneID)
					
					var pfTxt []string
					if err := json.Unmarshal([]byte(scene.FilenamesArr), &pfTxt); err == nil {
						pfTxt = append(pfTxt, fl.Filename)
						if tmp, err := json.Marshal(pfTxt); err == nil {
							scene.FilenamesArr = string(tmp)
							scene.Save()
							scene.UpdateStatus()
						}
					}
					
					tlog.Infof("Auto-matched funscript %v to scene %v", fl.Filename, scene.SceneID)
				}
			}

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
		if !files[i].IsDir() && funk.Contains(getVideoExtensions(), strings.ToLower(filepath.Ext(files[i].Name))) {
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
