package migrations

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
	"github.com/mozillazg/go-slugify"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
	"github.com/xbapps/xbvr/pkg/tasks"
	"gopkg.in/gormigrate.v1"
	"gopkg.in/resty.v1"
)

type RequestSceneList struct {
	DlState      optional.String   `json:"dlState"`
	Limit        optional.Int      `json:"limit"`
	Offset       optional.Int      `json:"offset"`
	IsAvailable  optional.Bool     `json:"isAvailable"`
	IsAccessible optional.Bool     `json:"isAccessible"`
	IsWatched    optional.Bool     `json:"isWatched"`
	Lists        []optional.String `json:"lists"`
	Sites        []optional.String `json:"sites"`
	Tags         []optional.String `json:"tags"`
	Cast         []optional.String `json:"cast"`
	Cuepoint     []optional.String `json:"cuepoint"`
	Volume       optional.Int      `json:"volume"`
	Released     optional.String   `json:"releaseMonth"`
	Sort         optional.String   `json:"sort"`
}

func (i *RequestSceneList) ToJSON() string {
	b, err := json.Marshal(i)
	if err != nil {
		return ""
	}
	return string(b)
}

func Migrate() {
	db, _ := models.GetDB()

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "0001",
			Migrate: func(tx *gorm.DB) error {
				return tx.
					AutoMigrate(&models.Scene{}).
					AutoMigrate(&models.SceneCuepoint{}).
					AutoMigrate(&models.Actor{}).
					AutoMigrate(&models.Tag{}).
					AutoMigrate(&models.File{}).
					AutoMigrate(&models.Volume{}).
					AutoMigrate(&models.History{}).
					AutoMigrate(&models.Site{}).
					AutoMigrate(&models.KV{}).Error
			},
		},
		{
			ID: "0002",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					VideoAvgFrameRateVal float64
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0003",
			Migrate: func(tx *gorm.DB) error {
				var files []models.File
				tx.Model(&files).Find(&files)

				for i := range files {
					err := files[i].CalculateFramerate()
					if err == nil {
						files[i].Save()
					}
				}
				return nil
			},
		},
		{
			ID: "0004",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					NeedsUpdate bool
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0005",
			Migrate: func(tx *gorm.DB) error {
				type Volume struct {
					Type     string
					Metadata string
				}
				return tx.AutoMigrate(Volume{}).Error
			},
		},
		{
			ID: "0006",
			Migrate: func(tx *gorm.DB) error {
				var volumes []models.Volume
				tx.Model(&volumes).Find(&volumes)

				for i := range volumes {
					if volumes[i].Type == "" {
						volumes[i].Type = "local"
						volumes[i].Save()
					}
				}
				return nil
			},
		},
		{
			ID: "0007",
			Migrate: func(tx *gorm.DB) error {
				type Site struct {
					AvatarURL string
				}
				return tx.AutoMigrate(Site{}).Error
			},
		},
		// 0.3.0-beta.12
		{
			// VRCONK has changed scene numbering schema, so it needs to be flushed
			ID: "0007-flush-vrconk",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				db.Where("site = ?", "VRCONK").Find(&scenes)

				for _, obj := range scenes {
					files, _ := obj.GetFiles()
					for _, file := range files {
						file.SceneID = 0
						file.Save()
					}
				}

				return db.Where("site = ?", "VRCONK").Delete(&models.Scene{}).Error
			},
		},
		{
			ID: "0008-create-playlist-table",
			Migrate: func(tx *gorm.DB) error {
				type Playlist struct {
					ID        uint `gorm:"primary_key"`
					CreatedAt time.Time
					UpdatedAt time.Time

					Name         string
					Ordering     int
					IsSystem     bool
					IsDeoEnabled bool
					IsSmart      bool
					SearchParams string `sql:"type:text;"`
				}
				return tx.AutoMigrate(Playlist{}).Error
			},
		},
		{
			ID: "0009-create-default-lists",
			Migrate: func(tx *gorm.DB) error {
				list := RequestSceneList{
					IsAvailable:  optional.NewBool(true),
					IsAccessible: optional.NewBool(true),
					Sort:         optional.NewString("release_date_desc"),
				}

				listDefault := models.Playlist{
					Name:         "Default",
					IsSystem:     true,
					IsSmart:      true,
					IsDeoEnabled: false,
					Ordering:     -100,
					SearchParams: list.ToJSON(),
				}
				listDefault.Save()

				list = RequestSceneList{
					IsAvailable:  optional.NewBool(true),
					IsAccessible: optional.NewBool(true),
					Sort:         optional.NewString("release_date_desc"),
				}
				listDeoRecent := models.Playlist{
					Name:         "Recent",
					IsSystem:     true,
					IsSmart:      true,
					IsDeoEnabled: true,
					Ordering:     -49,
					SearchParams: list.ToJSON(),
				}
				listDeoRecent.Save()

				list = RequestSceneList{
					IsAvailable:  optional.NewBool(true),
					IsAccessible: optional.NewBool(true),
					Lists:        []optional.String{optional.NewString("favourite")},
					Sort:         optional.NewString("release_date_desc"),
				}
				listDeoFav := models.Playlist{
					Name:         "Favourite",
					IsSystem:     true,
					IsSmart:      true,
					IsDeoEnabled: true,
					Ordering:     -48,
					SearchParams: list.ToJSON(),
				}
				listDeoFav.Save()

				list = RequestSceneList{
					IsAvailable:  optional.NewBool(true),
					IsAccessible: optional.NewBool(true),
					Lists:        []optional.String{optional.NewString("watchlist")},
					Sort:         optional.NewString("release_date_desc"),
				}
				listDeoWatch := models.Playlist{
					Name:         "Watchlist",
					IsSystem:     true,
					IsSmart:      true,
					IsDeoEnabled: true,
					Ordering:     -47,
					SearchParams: list.ToJSON(),
				}
				listDeoWatch.Save()

				return nil
			},
		},
		{
			ID: "0010-preview-flag",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					HasVideoPreview bool `json:"has_preview" gorm:"default:false"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0011-upgrade-ffmpeg",
			Migrate: func(tx *gorm.DB) error {
				ffmpegPath := filepath.Join(common.BinDir, "ffmpeg")
				ffprobePath := filepath.Join(common.BinDir, "ffprobe")
				if runtime.GOOS == "windows" {
					ffmpegPath = ffmpegPath + ".exe"
					ffprobePath = ffprobePath + ".exe"
				}

				os.Remove(ffmpegPath)
				os.Remove(ffprobePath)
				return nil
			},
		},
		{
			ID: "0012-preview-flag",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					TotalFileSize int64 `json:"total_file_size" gorm:"default:0"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			// WetVR has changed scene numbering schema, so it needs to be flushed
			ID: "0013-flush-wetvr",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				db.Where("site = ?", "WetVR").Find(&scenes)

				for _, obj := range scenes {
					files, _ := obj.GetFiles()
					for _, file := range files {
						file.SceneID = 0
						file.Save()
					}
				}

				return db.Where("site = ?", "WetVR").Delete(&models.Scene{}).Error
			},
		},
		{
			// Migrate EvilEyeVR to VRPorn scraper. Will cause new scene IDs
			ID: "0014-evileye-to-vrporn",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				db.Where("site = ?", "EvilEyeVR").Find(&scenes)

				for _, obj := range scenes {
					files, _ := obj.GetFiles()
					for _, file := range files {
						file.SceneID = 0
						file.Save()
					}
				}

				return db.Where("site = ?", "EvilEyeVR").Delete(&models.Scene{}).Error
			},
		},
		{
			ID: "0015-scene-edits",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.Action{}).Error
			},
		},
		{
			ID: "0016-action-value-size",
			Migrate: func(tx *gorm.DB) error {
				if models.GetDBConn().Driver == "mysql" {
					tx.Model(&models.Action{}).Exec("RENAME TABLE actions TO actions_old")
				} else {
					tx.Model(&models.Action{}).Exec("ALTER TABLE actions RENAME TO actions_old")
				}
				tx.AutoMigrate(&models.Action{})
				return tx.Model(&models.Action{}).Exec("INSERT INTO actions SELECT * FROM actions_old").Error
			},
		},
		{
			ID: "0017-scene-multipart",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					IsMultipart bool `json:"is_multipart"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0018-added-file-type",
			Migrate: func(tx *gorm.DB) error {
				err := tx.AutoMigrate(&models.File{}).Error

				var files []models.File
				db.Find(&files)
				for _, file := range files {
					file.Type = "video"
					file.Save()
				}
				return err
			},
		},
		{
			ID: "0019-scene-is-scripted",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					IsScripted bool `json:"is_scripted" gorm:"default:false"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0020-scene-total-watch-time",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					TotalWatchTime int `json:"total_watch_time" gorm:"default:0"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0021-change-mkx200-projection",
			Migrate: func(tx *gorm.DB) error {
				err := tx.AutoMigrate(&models.File{}).Error

				filenameSeparator := regexp.MustCompile("[ _.-]+")
				var files []models.File
				db.Find(&files)
				for _, file := range files {
					if file.VideoProjection == "180_sbs" {
						nameparts := filenameSeparator.Split(strings.ToLower(file.Filename), -1)
						for _, part := range nameparts {
							if part == "mkx200" {
								file.VideoProjection = "mkx200"
								file.Save()
								break
							}
						}
					}
				}
				return err
			},
		},
		{
			ID: "0022-change-video-projection",
			Migrate: func(tx *gorm.DB) error {
				err := tx.AutoMigrate(&models.File{}).Error

				filenameSeparator := regexp.MustCompile("[ _.-]+")
				var files []models.File
				db.Find(&files)
				for _, file := range files {
					if file.VideoProjection == "180_sbs" {
						nameparts := filenameSeparator.Split(strings.ToLower(file.Filename), -1)
						for _, part := range nameparts {
							if part == "mkx200" || part == "mkx220" || part == "vrca220" {
								file.VideoProjection = part
								file.Save()
								break
							}
						}
					}
				}
				return err
			},
		},
		{
			ID: "0023-file-has-heatmap",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					HasHeatmap bool `json:"has_heatmap" gorm:"default:false"`
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0024-file-is-selected-script",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					IsSelectedScript bool `json:"is_selected_script" gorm:"default:false"`
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0025-file-is-exported",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					IsExported bool `json:"is_exported" gorm:"default:false"`
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0024-drop-actions-old",
			Migrate: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE IF EXISTS actions_old").Error
			},
		},
		{
			ID: "0025-playlist-add-dlstate",
			Migrate: func(tx *gorm.DB) error {
				var playlists []models.Playlist
				db.Find(&playlists)
				for _, playlist := range playlists {
					if playlist.IsSystem {
						var jsonResult RequestSceneList
						json.Unmarshal([]byte(playlist.SearchParams), &jsonResult)

						hasChanged := false
						if !jsonResult.DlState.Present() {
							jsonResult.DlState = optional.NewString("available")
							hasChanged = true
						}
						if !jsonResult.Volume.Present() {
							jsonResult.Volume = optional.NewInt(0)
							hasChanged = true
						}
						if hasChanged {
							playlist.SearchParams = jsonResult.ToJSON()
							playlist.Save()
						}
					}
				}
				return nil
			},
		},
		{
			ID: "0026-playlist-set-lists",
			Migrate: func(tx *gorm.DB) error {
				var playlists []models.Playlist
				db.Find(&playlists)
				for _, playlist := range playlists {
					if playlist.IsSystem {
						var jsonResult RequestSceneList
						json.Unmarshal([]byte(playlist.SearchParams), &jsonResult)

						if jsonResult.Lists == nil {
							jsonResult.Lists = []optional.String{}
							playlist.SearchParams = jsonResult.ToJSON()
							playlist.Save()
						}
					}
				}
				return nil
			},
		},
		{
			// VRBangers have removed scene numbering schema, so scene IDs need to be changed
			ID: "0027-fix-vrbangers-ids",
			Migrate: func(tx *gorm.DB) error {
				// old slug -> new slug
				slugMapping := map[string]string{
					"ayumis-first-time-2": "ayumis-first-time",
				}

				// site -> slug -> id
				newScenes := map[string]map[string]string{}
				newSceneId := func(site string, slug string) (string, error) {
					mapping, ok := newScenes[site]
					if !ok {
						mapping = map[string]string{}
						queryParams := "page=1&type=videos&sort=latest&show_custom_video=1&bonus-video=1&limit=1000"
						url := fmt.Sprintf("https://content.%s.com/api/content/v1/videos?%s", strings.ToLower(site), queryParams)
						r, err := resty.R().SetHeader("User-Agent", scrape.UserAgent).Get(url)
						if err != nil {
							return "", err
						}
						items := gjson.Get(r.String(), "data.items")
						if !items.Exists() {
							return "", fmt.Errorf("invalid response from %s API: no scenes found", site)
						}
						for _, scene := range items.Array() {
							id, slug := scene.Get("id").String(), scene.Get("slug").String()
							if id == "" || slug == "" {
								return "", fmt.Errorf("invalid response from %s API: no id or slug found", site)
							}
							mapping[slug] = slugify.Slugify(site) + "-" + id[15:]
						}
						newScenes[site] = mapping
					}
					return mapping[slug], nil
				}

				var scenes []models.Scene
				err := tx.Where("studio = ?", "VRBangers").Find(&scenes).Error
				if err != nil {
					return err
				}
				for _, scene := range scenes {
					trimmedURL := strings.TrimRight(scene.SceneURL, "/")
					dir, base := path.Split(trimmedURL)
					slug, ok := slugMapping[base]
					if !ok {
						slug = slugify.Slugify(base)
					}

					sceneID, err := newSceneId(scene.Site, slug)
					if err != nil {
						return err
					}
					if sceneID == "" {
						common.Log.Warnf("Could not update scene %s", scene.SceneID)
						continue
					}

					// update all actions referring to this scene by its scene_id
					err = tx.Model(&models.Action{}).Where("scene_id = ?", scene.SceneID).Update("scene_id", sceneID).Error
					if err != nil {
						return err
					}

					// update the scene itself
					// with trailing slash for consistency with scraped data, to avoid these scenes being re-scraped
					scene.SceneURL = dir + slug + "/"
					scene.SceneID = sceneID
					err = tx.Save(&scene).Error
					if err != nil {
						return err
					}
				}

				// since scenes have new IDs, we need to re-index them
				tasks.SearchIndex()

				return nil
			},
		},
		{
			// SLR/RealJam Titles containing ":" & "?" creates invalid filenames breaks automatching. fix filenames changing to _
			ID: "0029-fix-slr-rj-filenames",
			Migrate: func(tx *gorm.DB) error {
				filenameRegEx := regexp.MustCompile(`[:?]|( & )|( \\u0026 )`)
				var scenes []models.Scene
				err := tx.Where("filenames_arr LIKE ?", "%:%").Or("filenames_arr LIKE ?", "%?%").Or("filenames_arr LIKE ?", "%\\u0026%").Find(&scenes).Error
				if err != nil {
					return err
				}

				for _, scene := range scenes {
					scene.FilenamesArr = filenameRegEx.ReplaceAllString(scene.FilenamesArr, "_")
					err = tx.Save(&scene).Error
					if err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			// VRConk is now using VRBangers code. renumbering scenes
			ID: "0030-fix-vrconk-ids",
			Migrate: func(tx *gorm.DB) error {
				// old slug -> new slug
				slugMapping := map[string]string{
					"vrconk-scene": "vrconk-scene-0",
				}

				// site -> slug -> id
				newScenes := map[string]map[string]string{}
				newSceneId := func(site string, slug string) (string, error) {
					mapping, ok := newScenes[site]
					if !ok {
						mapping = map[string]string{}
						queryParams := "page=1&type=videos&sort=latest&show_custom_video=1&bonus-video=1&limit=1000"
						url := fmt.Sprintf("https://content.%s.com/api/content/v1/videos?%s", strings.ToLower(site), queryParams)
						r, err := resty.R().SetHeader("User-Agent", scrape.UserAgent).Get(url)
						if err != nil {
							return "", err
						}
						items := gjson.Get(r.String(), "data.items")
						if !items.Exists() {
							return "", fmt.Errorf("invalid response from %s API: no scenes found", site)
						}
						for _, scene := range items.Array() {
							id, slug := scene.Get("id").String(), scene.Get("slug").String()
							if id == "" || slug == "" {
								return "", fmt.Errorf("invalid response from %s API: no id or slug found", site)
							}
							mapping[slug] = slugify.Slugify(site) + "-" + id[15:]
						}
						newScenes[site] = mapping
					}
					return mapping[slug], nil
				}

				var scenes []models.Scene
				err := tx.Where("studio = ?", "VRCONK").Find(&scenes).Error
				if err != nil {
					return err
				}
				for _, scene := range scenes {
					trimmedURL := strings.TrimRight(scene.SceneURL, "/")
					dir, base := path.Split(trimmedURL)
					slug, ok := slugMapping[base]
					if !ok {
						slug = slugify.Slugify(base)
					}

					sceneID, err := newSceneId(scene.Site, slug)
					if err != nil {
						return err
					}
					if sceneID == "" {
						common.Log.Warnf("Could not update scene %s", scene.SceneID)
						continue
					}

					// update all actions referring to this scene by its scene_id
					err = tx.Model(&models.Action{}).Where("scene_id = ?", scene.SceneID).Update("scene_id", sceneID).Error
					if err != nil {
						return err
					}

					// update the scene itself
					// with trailing slash for consistency with scraped data, to avoid these scenes being re-scraped
					scene.SceneURL = dir + slug + "/"
					scene.SceneID = sceneID
					err = tx.Save(&scene).Error
					if err != nil {
						return err
					}
				}

				// since scenes have new IDs, we need to re-index them
				tasks.SearchIndex()

				return nil
			},
		},
		{
			// Moving VRPFilms to SLR
			ID: "0031-vrpfilms-to-slr",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				db.Where("site = ?", "VRP Films").Find(&scenes)

				for _, obj := range scenes {
					files, _ := obj.GetFiles()
					for _, file := range files {
						file.SceneID = 0
						file.Save()
					}
				}

				return db.Where("site = ?", "VRP Films").Delete(&models.Scene{}).Error
			},
		},
		{
			ID: "0032-move-tngf-to-vrphub",
			Migrate: func(tx *gorm.DB) error {
				// old slug -> new slug
				slugMapping := map[string]string{
					"2022-05-18": "2022-05-17",
				}

				vrporn := [...]string{"2020-04-07", "2020-04-15", "2021-01-09", "2021-01-25", "2021-01-31", "2021-02-21", "2021-03-27", "2021-04-10", "2021-05-16", "2021-06-15", "2021-06-21", "2021-07-05", "2021-07-18", "2021-07-31", "2021-08-17", "2021-08-29", "2021-10-09", "2021-10-11", "2021-10-19", "2021-11-20", "2021-11-24", "2021-11-25"}
				vrphub := [...]string{"2020-03-27", "2020-04-04", "2021-01-08", "2021-01-22", "2021-01-30", "2021-02-12", "2021-03-26", "2021-04-09", "2021-05-07", "2021-06-04", "2021-06-18", "2021-07-02", "2021-07-16", "2021-07-30", "2021-08-13", "2021-08-27", "2021-09-10", "2021-09-24", "2021-10-08", "2021-10-22", "2021-11-05", "2021-11-19"}

				// site -> slug -> id
				newScenes := map[string]map[string]string{}
				newScenes2 := map[string]map[string]string{}
				newSceneId := func(site string, slug string) (string, string, error) {
					mapping2, ok := newScenes2[site]
					mapping1, ok := newScenes[site]
					if !ok {
						mapping1 = map[string]string{}
						mapping2 = map[string]string{}
						//            queryParams := "https://vrphub.com/wp-json/wp/v2/posts?categories=2630&per_page=100"
						url := fmt.Sprintf("https://vrphub.com/wp-json/wp/v2/posts?categories=2630&per_page=100&order=asc")
						r, err := resty.R().SetHeader("User-Agent", scrape.UserAgent).Get(url)
						if err != nil {
							return "", "", err
						}
						items := gjson.Get(r.String(), "@this")
						if !items.Exists() {
							return "", "", fmt.Errorf("invalid response from %s API: no scenes found", site)
						}
						for _, scene := range items.Array() {
							id, slug, slug2 := scene.Get("id").String(), scene.Get("date").String(), scene.Get("link").String()
							if id == "" || slug == "" || slug2 == "" {
								return "", "", fmt.Errorf("invalid response from %s API: no id or slug found", site)
							}
							tmpDate, err := time.Parse(time.RFC3339, slug+"+00:00")
							if err == nil {
								slug = tmpDate.Format("2006-01-02")
							}
							mapping1[slug] = slugify.Slugify(site) + "-" + id
							mapping2[slug] = slug2
						}
						newScenes[site] = mapping1
						newScenes2[site] = mapping2
					}
					return mapping1[slug], mapping2[slug], nil
				}

				var scenes []models.Scene
				err := tx.Where("site = ?", "Tonight's Girlfriend VR").Find(&scenes).Error
				if err != nil {
					return err
				}
				for i, v := range vrporn {
					for _, scene := range scenes {
						if scene.ReleaseDateText == v {
							scene.ReleaseDateText = vrphub[i]
							err = tx.Save(&scene).Error
							if err != nil {
								return err
							}
							//							continue
						}
					}
				}

				for _, scene := range scenes {
					base := strings.TrimSpace(scene.ReleaseDateText)
					//					slug2 := slugMapping[strings.TrimSpace(scene.SceneURL)]
					//        dir, base := path.Split(trimmedURL)
					slug, ok := slugMapping[base]
					if !ok {
						slug = base
						//						slug2 = strings.TrimSpace(scene.SceneURL)
					}

					sceneID, urllink, err := newSceneId(scene.Site, slug)
					if err != nil {
						return err
					}
					if sceneID == "" {
						common.Log.Warnf("Could not update scene %s", scene.SceneID)
						continue
					}

					// update all actions referring to this scene by its scene_id
					err = tx.Model(&models.Action{}).Where("scene_id = ?", scene.SceneID).Update("scene_id", sceneID).Update("scene_url", urllink).Error
					if err != nil {
						return err
					}

					// update the scene itself
					// with trailing slash for consistency with scraped data, to avoid these scenes being re-scraped
					scene.SceneURL = urllink
					scene.SceneID = sceneID
					err = tx.Save(&scene).Error
					if err != nil {
						return err
					}
					common.Log.Infof("Updated scene %s", scene.SceneID)
				}

				// since scenes have new IDs, we need to re-index them
				tasks.SearchIndex()

				return nil
			},
		},
	})

	if err := m.Migrate(); err != nil {
		common.Log.Fatalf("Could not migrate: %v", err)
	}
	common.Log.Printf("Migration did run successfully")

	db.Close()
}
