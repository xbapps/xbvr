package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

func extractDebridFileID(p string) string {
	if strings.Contains(p, "||") {
		parts := strings.Split(p, "||")
		return parts[1]
	}
	return p
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
			case "debridlink":
				scanDebridLink(vol[i], db, tlog)
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

					if err == gorm.ErrRecordNotFound || fl.VolumeID == 0 || fl.Size != f.Size() || fl.OsHash == "" || fl.VideoWidth == 0 || fl.VideoHeight == 0 || fl.VideoDuration == 0 {
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

func scanDebridLink(vol models.Volume, db *gorm.DB, tlog *logrus.Entry) {
	// Create HTTP client with authorization header
	client := &http.Client{}

	// First verify account is valid
	httpReq, err := http.NewRequest("GET", "https://debrid-link.com/api/v2/account/infos", nil)
	if err != nil {
		vol.IsAvailable = false
		vol.Save()
		return
	}
	httpReq.Header.Add("Authorization", "Bearer "+vol.Metadata)

	// Make request to verify token
	httpResp, err := client.Do(httpReq)
	if err != nil {
		vol.IsAvailable = false
		vol.Save()
		return
	}
	defer httpResp.Body.Close()

	// Parse response
	var accountInfo struct {
		Success bool `json:"success"`
		Value   struct {
			Username string `json:"username"`
		} `json:"value"`
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&accountInfo); err != nil {
		vol.IsAvailable = false
		vol.Save()
		return
	}

	if !accountInfo.Success {
		vol.IsAvailable = false
		vol.Save()
		return
	}

	// Initialize concurrency control for ffprobe calls
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	processedFiles := 0

	// Get files list with pagination
	page := 0
	var currentFileID []string

	for {
		// Log the current page being scanned
		tlog.Infof("Debrid-Link: Scanning page %d", page)
		// Fetch files for current page
		listURL := fmt.Sprintf("https://debrid-link.com/api/v2/seedbox/list?perPage=100&page=%d", page)
		httpReq, err := http.NewRequest("GET", listURL, nil)
		if err != nil {
			break
		}
		httpReq.Header.Add("Authorization", "Bearer "+vol.Metadata)

		httpResp, err := client.Do(httpReq)
		if err != nil {
			break
		}

		// Parse response
		var filesResponse struct {
			Success bool `json:"success"`
			Value   []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				TotalSize int64  `json:"totalSize"`
				Files     []struct {
					ID              string `json:"id"`
					Name            string `json:"name"`
					Size            int64  `json:"size"`
					DownloadURL     string `json:"downloadUrl"`
					DownloadPercent int    `json:"downloadPercent"`
				} `json:"files"`
				Created int64 `json:"created"`
			} `json:"value"`
			Pagination struct {
				Page  int `json:"page"`
				Pages int `json:"pages"`
				Next  int `json:"next"`
			} `json:"pagination"`
		}

		if err := json.NewDecoder(httpResp.Body).Decode(&filesResponse); err != nil {
			httpResp.Body.Close()
			break
		}
		httpResp.Body.Close()

		if !filesResponse.Success {
			break
		}

		// Process files
		for _, torrent := range filesResponse.Value {
			for _, file := range torrent.Files {
				if funk.Contains(allowedVideoExt, strings.ToLower(filepath.Ext(file.Name))) && file.DownloadPercent == 100 {
					processedFiles++
					tlog.Infof("Debrid-Link: Processing file %d: torrent '%s' - file '%s'", processedFiles, torrent.Name, file.Name)
					// Use a friendly display path and store original file ID in the Path field using '||' as separator
					displayPath := "Debrid-Link (" + accountInfo.Value.Username + ")/Seedbox"
					var fl models.File
					err = db.Where("path LIKE ? AND filename = ?", "%||"+file.ID, file.Name).First(&fl).Error
					if err == gorm.ErrRecordNotFound {
						var newFile models.File
						db.Where("path LIKE ? AND filename = ?", "%||"+file.ID, file.Name).FirstOrCreate(&newFile)
						// Store file ID in a hidden format that doesn't affect display path
						newFile.Path = displayPath + "||" + file.ID
						newFile.Filename = file.Name
						newFile.VideoProjection = "180_sbs"
						newFile.Size = file.Size
						newFile.Type = "video"
						newFile.CreatedTime = time.Unix(torrent.Created, 0)
						newFile.UpdatedTime = time.Now()
						newFile.VolumeID = vol.ID
						newFile.Save()
						fl = newFile
					} else {
						// Update display path if needed and ensure filename is set
						fl.Path = displayPath + "||" + file.ID
						fl.Filename = file.Name
						fl.Save()
					}
					currentFileID = append(currentFileID, file.ID)

					// Spawn a goroutine to extract metadata via ffprobe in parallel if metadata is not already present
					if fl.VideoDuration > 0 {
						tlog.Infof("Debrid-Link: Skipping ffprobe for file '%s' as metadata already exists.", file.Name)
					} else {
						wg.Add(1)
						go func(downloadURL string, fileID uint, fileName string) {
							defer wg.Done()
							sem <- struct{}{}
							defer func() { <-sem }()

							tlog.Infof("Debrid-Link: Starting ffprobe for file '%s' (ID: %d)", fileName, fileID)

							// Create a context with timeout to prevent hanging
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()

							// Use a channel to signal completion
							done := make(chan bool, 1)
							var ffdata *ffprobe.ProbeData
							var ffErr error

							go func() {
								ffdata, ffErr = ffprobe.GetProbeData(downloadURL, 10*time.Second)
								done <- true
							}()

							// Wait for either completion or timeout
							select {
							case <-done:
								if ffErr != nil {
									tlog.Errorf("Debrid-Link: ffprobe error for file '%s': %v", fileName, ffErr)
									return
								}
							case <-ctx.Done():
								tlog.Errorf("Debrid-Link: ffprobe timeout for file '%s'", fileName)
								return
							}

							if ffdata == nil {
								tlog.Errorf("Debrid-Link: No ffprobe data for file '%s'", fileName)
								return
							}

							vs := ffdata.GetFirstVideoStream()
							if vs == nil {
								tlog.Errorf("Debrid-Link: No video stream found for file '%s'", fileName)
								return
							}

							updates := map[string]interface{}{}
							updates["video_width"] = vs.Width
							updates["video_height"] = vs.Height
							if vs.BitRate != "" {
								if br, err := strconv.Atoi(vs.BitRate); err == nil {
									updates["video_bit_rate"] = br
								}
							}
							if dur, err := strconv.ParseFloat(vs.Duration, 64); err == nil {
								updates["video_duration"] = dur
							} else if ffdata.Format.DurationSeconds > 0.0 {
								updates["video_duration"] = ffdata.Format.DurationSeconds
							}
							if vs.AvgFrameRate != "" {
								if fps, err := strconv.ParseFloat(vs.AvgFrameRate, 64); err == nil {
									updates["video_avg_frame_rate_val"] = fps
								}
							}

							err := db.Model(&models.File{}).Where("id = ?", fileID).Updates(updates).Error
							if err != nil {
								tlog.Errorf("Debrid-Link: Failed to update metadata for file '%s': %v", fileName, err)
							} else {
								tlog.Infof("Debrid-Link: Successfully updated metadata for file '%s'", fileName)
							}
						}(file.DownloadURL, fl.ID, file.Name)
					}
				}
			}
		}

		if filesResponse.Pagination.Next == -1 {
			break
		}
		page = filesResponse.Pagination.Next
	}

	// Wait for all parallel ffprobe metadata extraction routines to finish
	wg.Wait()

	// Check if local files are present in listing
	var scene models.Scene
	allFiles := vol.Files()
	for i := range allFiles {
		if !funk.ContainsString(currentFileID, extractDebridFileID(allFiles[i].Path)) {
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
	vol.Path = "Debrid-Link (" + accountInfo.Value.Username + ")"
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

// FixDebridLinkPaths updates all Debrid-Link file paths to use the correct format
func FixDebridLinkPaths() {
	tlog := log.WithFields(logrus.Fields{"task": "fix-paths"})
	tlog.Infof("Fixing Debrid-Link file paths")

	db, _ := models.GetDB()
	defer db.Close()

	// Get all volumes of type debridlink
	var volumes []models.Volume
	db.Where("type = ?", "debridlink").Find(&volumes)

	for _, vol := range volumes {
		// Get account info to construct the correct path
		client := &http.Client{}
		httpReq, err := http.NewRequest("GET", "https://debrid-link.com/api/v2/account/infos", nil)
		if err != nil {
			tlog.Errorf("Error creating request: %v", err)
			continue
		}
		httpReq.Header.Add("Authorization", "Bearer "+vol.Metadata)

		httpResp, err := client.Do(httpReq)
		if err != nil {
			tlog.Errorf("Error getting account info: %v", err)
			continue
		}

		var accountInfo struct {
			Success bool `json:"success"`
			Value   struct {
				Username string `json:"username"`
			} `json:"value"`
		}

		if err := json.NewDecoder(httpResp.Body).Decode(&accountInfo); err != nil {
			httpResp.Body.Close()
			tlog.Errorf("Error decoding response: %v", err)
			continue
		}
		httpResp.Body.Close()

		if !accountInfo.Success {
			tlog.Errorf("API returned error for volume %d", vol.ID)
			continue
		}

		// Construct the new base path
		basePath := "Debrid-Link (" + accountInfo.Value.Username + ")/Seedbox"

		// Get all files for this volume
		var files []models.File
		db.Where("volume_id = ?", vol.ID).Find(&files)

		updatedCount := 0
		for _, file := range files {
			// Extract the file ID from the current path
			fileID := ""
			if strings.Contains(file.Path, "||") {
				parts := strings.Split(file.Path, "||")
				if len(parts) > 1 {
					fileID = parts[1]

					// Update the path to the new format
					newPath := basePath + "||" + fileID
					if file.Path != newPath {
						file.Path = newPath
						db.Save(&file)
						updatedCount++
					}
				}
			}
		}

		tlog.Infof("Updated %d files for volume %d", updatedCount, vol.ID)
	}

	tlog.Infof("Finished fixing Debrid-Link file paths")
}
