package migrations

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
	"github.com/mozillazg/go-slugify"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
	"github.com/xbapps/xbvr/pkg/tasks"
	"gopkg.in/gormigrate.v1"
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
			ID: "0038-edits-applied",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					EditsApplied bool `json:"edits_applied" gorm:"default:false"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0042-file-oshash",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					OsHash string `json:"oshash"`
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0043-actor-akas",
			Migrate: func(tx *gorm.DB) error {
				return tx.
					AutoMigrate(&models.Aka{}).Error
			},
		},
		{
			ID: "0043-ActorsAvailCount",
			Migrate: func(tx *gorm.DB) error {
				// auto migrate actor table to add avail_count column
				return tx.AutoMigrate(&models.Actor{}).Error
			},
		},
		{
			ID: "0044-Trailers-scenes",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					TrailerType   string `json:"trailer_type" xbvrbackup:"trailer_type"`
					TrailerSource string `gorm:"size:1000" json:"trailer_source" xbvrbackup:"trailer_source"`
					Trailerlist   bool   `json:"trailerlist" gorm:"default:false" xbvrbackup:"trailerlist"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0048-Add-Index-To-Cuepoints",
			Migrate: func(tx *gorm.DB) error {
				type SceneCuepoint struct {
					SceneID uint `gorm:"index" json:"-" xbvrbackup:"-"`
				}
				return tx.AutoMigrate(SceneCuepoint{}).Error
			},
		},
		{
			ID: "0050-members-url",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					MemberURL string `json:"members_url" xbvrbackup:"members_url"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0049-Add-Is_Hidden-To-Cuepoints",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					IsHidden bool `json:"is_hidden" gorm:"default:false" xbvrbackup:"is_hidden"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0051-heresphere-cuepoints",
			Migrate: func(tx *gorm.DB) error {
				type SceneCuepoint struct {
					SceneID uint    `gorm:"index" json:"-" xbvrbackup:"-"`
					TimeEnd float64 `json:"time_end" xbvrbackup:"time_end"`
					Track   uint    `json:"track" xbvrbackup:"track"`
					Rating  float64 `json:"rating" xbvrbackup:"rating"`
				}

				return tx.AutoMigrate(&SceneCuepoint{}).Error
			},
		},
		{
			ID: "0052-scene-index-on-files-for-feature-filters",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					SceneID uint `gorm:"index" json:"scene_id" xbvrbackup:"-"`
				}
				return tx.AutoMigrate(File{}).Error
			},
		},
		{
			ID: "0053-add-legacy-scene-id-to-save-old-scene_ids",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					LegacySceneID string `json:"legacy_scene_id" xbvrbackup:"legacy_scene_id"`
					ScraperId     string `json:"scraper_id" xbvrbackup:"scraper_id"`
				}
				return tx.AutoMigrate(Scene{}).Error
			},
		},
		{
			ID: "0056-Subscribed-Fields",
			Migrate: func(tx *gorm.DB) error {
				type Site struct {
					Subscribed bool `json:"subscribed" xbvrbackup:"subscribed"`
				}
				type Scene struct {
					IsSubscribed bool `json:"is_subscribed" gorm:"default:false"`
				}
				err := tx.AutoMigrate(Site{}).Error
				if err == nil {
					err = tx.AutoMigrate(Scene{}).Error
				}
				return err
			},
		},
		{
			ID: "0057-Tag-groups",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.TagGroup{}).Error
			},
		},
		{
			ID: "0058-Change-NewValue-Column-To-Text",
			Migrate: func(tx *gorm.DB) error {
				var err error
				switch tx.Dialect().GetName() {
				case "sqlite3":
					err = tx.Model(&models.Action{}).Exec("ALTER TABLE actions RENAME TO actions_old2").Error
				case "mysql":
					err = tx.Model(&models.Action{}).Exec("RENAME TABLE actions TO actions_old2").Error
				}
				if err != nil {
					return err
				}
				tx.AutoMigrate(&models.Action{})
				err = tx.Model(&models.Action{}).Exec("INSERT INTO actions SELECT * FROM actions_old2").Error
				if err != nil {
					return err
				}
				return tx.Exec("DROP TABLE IF EXISTS actions_old2").Error
			},
		},

		// ===============================================================================================
		// Put DB Schema migrations above this line and migrations that rely on the updated schema below
		// ===============================================================================================

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

						r, err := resty.New().R().SetHeader("User-Agent", scrape.UserAgent).Get(url)
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
						r, err := resty.New().R().SetHeader("User-Agent", scrape.UserAgent).Get(url)
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
			ID: "0032-fix-filters-with-playlist",
			Migrate: func(tx *gorm.DB) error {
				var playlists []models.Playlist
				db.Find(&playlists)
				for _, playlist := range playlists {
					var jsonResult RequestSceneList
					json.Unmarshal([]byte(playlist.SearchParams), &jsonResult)

					if jsonResult.Cast == nil {
						jsonResult.Cast = []optional.String{}
					}
					if jsonResult.Sites == nil {
						jsonResult.Sites = []optional.String{}
					}
					if jsonResult.Tags == nil {
						jsonResult.Tags = []optional.String{}
					}
					if jsonResult.Cuepoint == nil {
						jsonResult.Cuepoint = []optional.String{}
					}

					playlist.SearchParams = jsonResult.ToJSON()
					playlist.Save()
				}
				return nil
			},
		},
		{
			ID: "0033a-move-tngf-to-tngf",
			Migrate: func(tx *gorm.DB) error {

				// seed information old date -> new date -> new sceneid/url
				vrporn := [...]string{"2020-04-07", "2020-04-15", "2021-01-09", "2021-01-25", "2021-01-31", "2021-02-21", "2021-03-27", "2021-04-10", "2021-04-23", "2021-05-16", "2021-05-21", "2021-06-15", "2021-06-21", "2021-07-05", "2021-07-18", "2021-07-31", "2021-08-17", "2021-08-29", "2021-10-09", "2021-10-11", "2021-10-19", "2021-11-20", "2021-11-24", "2021-11-25"}
				tngf := [...]string{"2020-03-27", "2020-04-03", "2021-01-08", "2021-01-22", "2021-01-29", "2021-02-12", "2021-03-25", "2021-04-08", "2021-04-22", "2021-05-06", "2021-05-20", "2021-06-03", "2021-06-17", "2021-07-01", "2021-07-15", "2021-07-29", "2021-08-12", "2021-08-26", "2021-09-09", "2021-09-23", "2021-10-07", "2021-10-21", "2021-11-04", "2021-11-19"}
				tngf_json := `[{"date":"2020-03-27","sceneid":"tonight-s-girlfriend-vr-25906","sceneurl":"https://www.tonightsgirlfriend.com/scene/cherie-deville-always-satisfies-her-clients-and-fans-however-they-want-25906"},{"date":"2020-04-03","sceneid":"tonight-s-girlfriend-vr-25913","sceneurl":"https://www.tonightsgirlfriend.com/scene/brooklyn-gray-fucks-for-cash-in-vr-25913"},{"date":"2020-04-17","sceneid":"tonight-s-girlfriend-vr-25939","sceneurl":"https://www.tonightsgirlfriend.com/scene/kenna-james-gives-her-fan-what-he-wants-25939"},{"date":"2020-04-24","sceneid":"tonight-s-girlfriend-vr-25947","sceneurl":"https://www.tonightsgirlfriend.com/scene/jenna-j-ross-satisfies-her-super-fan-25947"},{"date":"2020-05-08","sceneid":"tonight-s-girlfriend-vr-25978","sceneurl":"https://www.tonightsgirlfriend.com/scene/ashley-lane-wears-sexy-lingerie-to-fuck-fan-in-hotel-room-25978"},{"date":"2020-06-12","sceneid":"tonight-s-girlfriend-vr-26032","sceneurl":"https://www.tonightsgirlfriend.com/scene/kenzie-madison-wears-sexy-lingerie-to-fuck-her-fan-26032"},{"date":"2020-07-03","sceneid":"tonight-s-girlfriend-vr-26068","sceneurl":"https://www.tonightsgirlfriend.com/scene/kenna-james-wears-lingerie-before-fucking-fan-26068"},{"date":"2020-12-11","sceneid":"tonight-s-girlfriend-vr-26342","sceneurl":"https://www.tonightsgirlfriend.com/scene/a-fan-gets-the-anna-claire-clouds-experience-hes-been-yearning-for-26342"},{"date":"2020-12-25","sceneid":"tonight-s-girlfriend-vr-26364","sceneurl":"https://www.tonightsgirlfriend.com/scene/fantasies-come-true-when-daisy-stone-visits-a-fan-for-a-memorable-night-26364"},{"date":"2021-01-08","sceneid":"tonight-s-girlfriend-vr-26379","sceneurl":"https://www.tonightsgirlfriend.com/scene/jamie-jett-pleases-a-fan-in-all-the-right-ways-that-only-a-pornstar-can-26379"},{"date":"2021-01-22","sceneid":"tonight-s-girlfriend-vr-26399","sceneurl":"https://www.tonightsgirlfriend.com/scene/a-fan-orders-himself-spencer-bradley-for-the-night-26399"},{"date":"2021-01-29","sceneid":"tonight-s-girlfriend-vr-26414","sceneurl":"https://www.tonightsgirlfriend.com/scene/emma-hix-shows-she-can-take-a-big-black-cock-with-ease-26414"},{"date":"2021-02-12","sceneid":"tonight-s-girlfriend-vr-26431","sceneurl":"https://www.tonightsgirlfriend.com/scene/aila-donovans-fan-gets-the-treatment-hes-been-yearning-for-26431"},{"date":"2021-02-26","sceneid":"tonight-s-girlfriend-vr-26447","sceneurl":"https://www.tonightsgirlfriend.com/scene/ivy-lebelle-stops-by-the-hotel-room-of-a-man-in-need-of-a-nice-big-ass-in-sexy-stockings-26447"},{"date":"2021-03-12","sceneid":"tonight-s-girlfriend-vr-26476","sceneurl":"https://www.tonightsgirlfriend.com/scene/quinn-wilde-fucks-her-fan-in-sexy-pink-lingerie-26476"},{"date":"2021-03-25","sceneid":"tonight-s-girlfriend-vr-26494","sceneurl":"https://www.tonightsgirlfriend.com/scene/kayley-gunner-takes-good-care-of-a-married-man-26494"},{"date":"2021-04-08","sceneid":"tonight-s-girlfriend-vr-26512","sceneurl":"https://www.tonightsgirlfriend.com/scene/petite-cutie-kylie-rocket-takes-care-of-a-big-man-with-a-big-package-26512"},{"date":"2021-04-22","sceneid":"tonight-s-girlfriend-vr-26532","sceneurl":"https://www.tonightsgirlfriend.com/scene/brooke-banner-gets-rough-fuck-from-thick-dick-fan-26532"},{"date":"2021-05-06","sceneid":"tonight-s-girlfriend-vr-26552","sceneurl":"https://www.tonightsgirlfriend.com/scene/casca-akashova-takes-care-of-a-married-man-in-need-of-some-affection-and-attention-26552"},{"date":"2021-05-20","sceneid":"tonight-s-girlfriend-vr-26578","sceneurl":"https://www.tonightsgirlfriend.com/scene/the-always-horny-adira-allure-hooks-up-with-her-super-fan-26578"},{"date":"2021-06-03","sceneid":"tonight-s-girlfriend-vr-26594","sceneurl":"https://www.tonightsgirlfriend.com/scene/sexy-tattooed-babe-penny-archer-hooks-up-with-her-fan-for-a-night-of-pleasure-26594"},{"date":"2021-06-17","sceneid":"tonight-s-girlfriend-vr-26617","sceneurl":"https://www.tonightsgirlfriend.com/scene/the-beautiful-emma-starletto-takes-on-a-married-man-in-his-hotel-26617"},{"date":"2021-07-01","sceneid":"tonight-s-girlfriend-vr-26640","sceneurl":"https://www.tonightsgirlfriend.com/scene/the-sexy-gia-derza-gives-her-fan-a-pornstar-experience-hell-never-forget-26640"},{"date":"2021-07-15","sceneid":"tonight-s-girlfriend-vr-26661","sceneurl":"https://www.tonightsgirlfriend.com/scene/the-sexy-angel-youngs-takes-special-care-of-a-client-in-need-26661"},{"date":"2021-07-29","sceneid":"tonight-s-girlfriend-vr-26679","sceneurl":"https://www.tonightsgirlfriend.com/scene/eliza-ibarra-sits-her-perfect-ass-on-the-cock-of-her-client-26679"},{"date":"2021-08-12","sceneid":"tonight-s-girlfriend-vr-26701","sceneurl":"https://www.tonightsgirlfriend.com/scene/mckenzie-lee-helps-her-fan-let-loose-relax-and-relieve-stress-the-best-way-one-can-26701"},{"date":"2021-08-26","sceneid":"tonight-s-girlfriend-vr-26720","sceneurl":"https://www.tonightsgirlfriend.com/scene/gianna-grey-finally-gets-together-with-her-longtime-fan-26720"},{"date":"2021-09-09","sceneid":"tonight-s-girlfriend-vr-29338","sceneurl":"https://www.tonightsgirlfriend.com/scene/scarlett-mae-shows-up-looking-delicious-and-sexy-in-lingerie-for-her-client-29338"},{"date":"2021-09-23","sceneid":"tonight-s-girlfriend-vr-29976","sceneurl":"https://www.tonightsgirlfriend.com/scene/penelope-kay-takes-care-of-her-big-dick-client-29976"},{"date":"2021-10-07","sceneid":"tonight-s-girlfriend-vr-30114","sceneurl":"https://www.tonightsgirlfriend.com/scene/diana-grace-dresses-in-sexy-red-lingerie-for-her-big-cock-client-30114"},{"date":"2021-10-21","sceneid":"tonight-s-girlfriend-vr-30629","sceneurl":"https://www.tonightsgirlfriend.com/scene/the-sexy-petite-pornstar-brooklyn-gray-shows-she-can-take-a-big-black-cock-anytime-30629"},{"date":"2021-11-04","sceneid":"tonight-s-girlfriend-vr-30648","sceneurl":"https://www.tonightsgirlfriend.com/scene/the-sexy-blake-blossom-puts-on-special-lingerie-just-for-her-client-30648"},{"date":"2021-11-19","sceneid":"tonight-s-girlfriend-vr-30677","sceneurl":"https://www.tonightsgirlfriend.com/scene/gorgeous-and-fit-babe-ana-foxxx-takes-care-of-her-fan-in-every-way-he-desires-30677"}]`

				var scenes []models.Scene
				err := tx.Where("site = ?", "Tonight's Girlfriend VR").Find(&scenes).Error
				if err != nil {
					return err
				}
				for _, scene := range scenes {
					for i, v := range vrporn {
						if scene.ReleaseDateText == v {
							scene.ReleaseDateText = tngf[i]
							err = tx.Save(&scene).Error
							if err != nil {
								return err
							}
							// common.Log.Infof("Updated scene %s", scene.SceneID)
						}
					}
				}

				var scenes_tngf []models.Scene
				err = tx.Where("site = ?", "Tonight's Girlfriend VR").Find(&scenes_tngf).Error
				if err != nil {
					return err
				}
				items := gjson.Get(tngf_json, "@this")
				for _, scene := range scenes_tngf {
					for _, scenejson := range items.Array() {
						if scene.ReleaseDateText == scenejson.Get("date").String() {
							scene.ReleaseDateText = scenejson.Get("date").String()
							scene.ReleaseDate, _ = time.Parse(time.RFC3339, scene.ReleaseDateText+"T00:00:00-04:00")
							sceneID := scenejson.Get("sceneid").String()
							scene.SceneURL = scenejson.Get("sceneurl").String()

							// update all actions referring to this scene by its scene_id
							err = tx.Model(&models.Action{}).Where("scene_id = ?", scene.SceneID).Update("scene_id", sceneID).Error
							if err != nil {
								return err
							}
							if scene.SceneID == sceneID && !(scene.IsAccessible || scene.IsAvailable) {
								err = tx.Delete(&scene).Where("scene_id = ?", sceneID).Error
								if err != nil {
									return err
								}
								continue
							}
							scene.SceneID = sceneID
							// update the scene itself
							err = tx.Save(&scene).Error
							if err != nil {
								return err
							}
							// common.Log.Infof("Updated scene %s", scene.SceneID)
						}
					}
				}

				// since scenes have new IDs, we need to re-index them
				tasks.SearchIndex()

				return nil
			},
		},
		{
			// rebuild search indexes with new fields
			ID: "034-rebuild-new-indexes",
			Migrate: func(d *gorm.DB) error {
				os.RemoveAll(common.IndexDirV2)
				os.MkdirAll(common.IndexDirV2, os.ModePerm)
				// rebuild asynchronously, no need to hold up startup, blocking the UI
				go func() {
					tasks.SearchIndex()
					tasks.CalculateCacheSizes()
				}()
				return nil
			},
		},
		{
			// some site, vrbangers & vrconk have blank covers, & vrbangers gallery images will not render due to double slashes ie .com//
			ID: "0035-fix-vrbangers-images",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				err := tx.Where("studio  LIKE ?", "VRBangers").Or("images LIKE ?", "%{\"url\":\"\",\"type\":\"gallery\",\"orientation\":\"\"}%").Find(&scenes).Error
				if err != nil {
					return err
				}

				for _, scene := range scenes {
					changed := false
					// check for a blank cover image and remove them
					if strings.Contains(scene.Images, ",{\"url\":\"\",\"type\":\"cover\",\"orientation\":\"\"}") {
						scene.Images = strings.ReplaceAll(scene.Images, ",{\"url\":\"\",\"type\":\"cover\",\"orientation\":\"\"}", "")
						changed = true
					}
					// remove double slashes from image url for VRBangers
					if scene.Studio == "VRBangers" && strings.Contains(scene.Images, ".com//") {
						scene.Images = strings.ReplaceAll(scene.Images, ".com//", ".com/")
						changed = true
					}
					if changed {
						err = tx.Save(&scene).Error
						if err != nil {
							return err
						}
					}
				}
				return nil
			},
		},
		{
			ID: "0036-fix-missing-cover-urls",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				err := tx.Where("cover_url=''").Find(&scenes).Error
				if err != nil {
					return err
				}

				var images []models.Image
				for _, scene := range scenes {
					changed := false
					if err := json.Unmarshal([]byte(scene.Images), &images); err == nil {
						for _, image := range images {
							if scene.CoverURL == "" && image.Type == "cover" {
								scene.CoverURL = image.URL
								changed = true
							}
						}
					}
					if changed {
						err = tx.Save(&scene).Error
						if err != nil {
							return err
						}
					}
				}
				return nil
			},
		},
		{
			ID: "0037-migrate-schedule-config",
			Migrate: func(d *gorm.DB) error {
				// get the old config values using the old json format
				var obj models.KV
				type oldConfigDef struct {
					Cron struct {
						ScrapeContentInterval int `json:"scrapeContentInt"`
						RescanLibraryInterval int `json:"rescanLibraryInt"`
					} `json:"cron"`
				}

				var oldConfig oldConfigDef
				err := d.Where(&models.KV{Key: "config"}).First(&obj).Error
				if err == nil {
					if err := json.Unmarshal([]byte(obj.Value), &oldConfig); err == nil {
						// update the new config
						config.Config.Cron.RescrapeSchedule.HourInterval = oldConfig.Cron.ScrapeContentInterval
						config.Config.Cron.RescanSchedule.HourInterval = oldConfig.Cron.RescanLibraryInterval
					}
				}
				// nicety to default scraping to a random start time, so everyone does not start scrapping on the hour, users can change it if they want
				ms := rand.Intn(59)
				config.Config.Cron.RescrapeSchedule.MinuteStart = ms
				config.SaveConfig()
				return nil
			},
		},
		{
			ID: "0039-title-size-change",
			Migrate: func(tx *gorm.DB) error {
				if models.GetDBConn().Driver == "mysql" {
					return tx.Model(&models.Scene{}).ModifyColumn("title", "varchar(1024)").Error
				}
				return nil
			},
		},
		{
			// perVRt change siteID
			ID: "0040-revert-pervrt",
			Migrate: func(tx *gorm.DB) error {
				return db.Model(&models.Scene{}).Where("site = ?", "perVRt/Terrible").Update("site", "perVRt").Error
			},
		},
		{
			// populate default cuepoint actions and positions
			ID: "0041-default-cuepoints",
			Migrate: func(tx *gorm.DB) error {
				var kv models.KV
				kv.Key = "cuepoints"
				db.Find(&kv)

				if kv.Value == "" {
					kv.Value = "{\"positions\":[\"standing\", \"sitting\", \"laying\", \"kneeling\"],\"actions\":[\"handjob\", \"blowjob\", \"doggy\", \"cowgirl\", \"revcowgirl\", \"missionary\", \"titfuck\", \"anal\", \"cumshot\", \"69\", \"facesit\"]}"
					kv.Save()
				}
				return nil
			},
		},
		{
			// R18 is being permanently shut down no later than January 31, 2023 - changes images to FANZA URLs
			ID: "0045-change-R18-to-FANZA",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				err := tx.Where("images LIKE ?", "%{\"url\":\"https://pics.r18.com%").Find(&scenes).Error
				if err != nil {
					return err
				}

				for _, scene := range scenes {
					changed := false
					// change all image URLs
					if strings.Contains(scene.Images, "pics.r18.com") {
						scene.Images = strings.ReplaceAll(scene.Images, "pics.r18.com", "pics.dmm.co.jp")
						changed = true
					}
					// change cover image URL
					if strings.Contains(scene.CoverURL, "pics.r18.com") {
						scene.CoverURL = strings.ReplaceAll(scene.CoverURL, "pics.r18.com", "pics.dmm.co.jp")
						changed = true
					}
					// change scene URL
					if strings.Contains(scene.SceneURL, "https://www.r18.com/videos/vod/movies/detail/-/") {
						scene.SceneURL = strings.ReplaceAll(scene.SceneURL, "https://www.r18.com/videos/vod/movies/detail/-/", "https://www.dmm.co.jp/digital/videoa/-/detail/=/c")
						changed = true
					}
					if changed {
						err = tx.Save(&scene).Error
						if err != nil {
							return err
						}
					}
				}
				return nil
			},
		},
		{
			ID: "00046-fix-orphaned-cuepoints",
			Migrate: func(tx *gorm.DB) error {
				return db.Where("scene_id is null").Delete(&models.SceneCuepoint{}).Error
			},
		},
		{
			ID: "0047-TrailerRules",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene

				err := tx.Find(&scenes).Error
				if err != nil {
					return err
				}
				for _, scene := range scenes {
					id := scene.SceneID[strings.LastIndex(scene.SceneID, "-")+1:]
					var re = regexp.MustCompile(`(?m).*.(\.com|\.net)\/`)
					baseUrl := re.FindAllString(scene.SceneURL, -1)

					if len(baseUrl) == 0 {
						continue
					}
					if baseUrl[0] == `https://www.sexlikereal.com/` {
						scene.TrailerType = "slr"
						scene.TrailerSource = `https://api.sexlikereal.com/virtualreality/video/id/` + id
					}
					if baseUrl[0] == `https://vrporn.com/` {
						scene.TrailerType = "scrape_html"
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"dl8-video source","extract_regex":"","content_base_url":"","record_path":"","content_path":"src","encoding_path":"","quality_path":"quality"}`
					}

					switch scene.Site {
					case "RealJam VR", "WetVR", "StasyQVR", "ZexyVR", "WankitNowVR", "SexBabesVR":
						scene.TrailerType = "deovr"
					case "BadoinkVR", "18VR", "BabeVR", "KinkVR", "VRCosplayX", "TmwVRnet", "SinsVR", "VirtualPorn", "Tonight's Girlfriend VR", "VRAllure",
						"VRHush", "VRLatina", "BaberoticaVR", "RealityLovers", "GroobyVR":
						scene.TrailerType = "scrape_html"
					case "BrasilVR", "Czech VR", "Czech VR Casting", "Czech VR Fetish", "MilfVR", "POVR Originals", "VR Intimacy", "WankzVR", "herPOVR":
						scene.TrailerType = "heresphere"
					case "VirtualRealPassion", "VirtualRealPorn", "VirtualRealTrans", "VirtualRealGay", "VirtualRealAmateurPorn", "CaribbeanCom VR":
						scene.TrailerType = "scrape_json"
					case "NaughtyAmerica VR", "LethalHardcoreVR", "WhorecraftVR":
						scene.TrailerType = "url"
					case "VRBangers", "VRBTrans", "VRBGay", "DarkRoomVR", "VirtualTaboo":
						scene.TrailerType = "load_json"
					}

					switch scene.Site {
					case "VRBangers", "VRBTrans", "VRBGay":
						base := strings.Split(scene.SceneURL, "/")
						if len(base) >= 5 {
							scene.TrailerSource = `{"scene_url":"https://content.` + base[2] + `/api/content/v1/videos/` + base[4] + `","html_element":"","extract_regex":"","content_base_url":"","record_path":"data.item.videoPlayerSources.trailer","content_path":"src","encoding_path":"","quality_path":"quality"}`
						}
					case "RealJam VR", "SexBabesVR":
						scene.TrailerSource = baseUrl[0] + "deovr/video/id/" + id
					case "ZexyVR", "WankitNowVR":
						scene.TrailerSource = baseUrl[0] + "deovr/video/" + id
					case "WetVR":
						scene.TrailerSource = strings.Replace(scene.SceneURL, "/video/", "/deovr/", 1)
					case "StasyQVR":
						scene.TrailerSource = "http://stasyqvr.com/deovr_feed/json/id/" + id
					case "TmwVRnet", "SinsVR", "VirtualPorn", "Tonight's Girlfriend VR":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"dl8-video source","extract_regex":"","content_base_url":"","record_path":"","content_path":"src","encoding_path":"","quality_path":"quality"}`
					case "VRAllure", "VRHush", "VRLatina":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"deo-video source","extract_regex":"","content_base_url":"https:","record_path":"","content_path":"src","encoding_path":"","quality_path":"quality"}`
					case "BaberoticaVR":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"video source","extract_regex":"","content_base_url":"https:","record_path":"","content_path":"src","encoding_path":"","quality_path":"data-res"}`
					case "BadoinkVR", "18VR", "BabeVR", "KinkVR", "VRCosplayX":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + "trailer" + `","html_element":"dl8-video source","extract_regex":"","content_base_url":"","record_path":"","content_path":"src","encoding_path":"","quality_path":"quality"}`
					case "DarkRoomVR":
						scene.TrailerSource = `{"scene_url":"https://darkroomvr.com/api/vrplayer/video/detail/` + id + `","html_element":"","extract_regex":"","content_base_url":"","record_path":"sources","content_path":"url","encoding_path":"","quality_path":"title"}`
					case "VirtualTaboo":
						scene.TrailerSource = `{"scene_url":"https://virtualtaboo.com/gizmo/videoinfo/` + id + `","html_element":"","extract_regex":"","content_base_url":"","record_path":"sources","content_path":"url","encoding_path":"","quality_path":"title"}`
					case "BrasilVR", "POVR Originals", "WankzVR", "herPOVR":
						scene.TrailerSource = baseUrl[0] + "heresphere/" + id
					case "Czech VR", "Czech VR Casting", "Czech VR Fetish", "VR Intimacy":
						var re = regexp.MustCompile(`detail-(\d*)`)
						internalId := re.FindStringSubmatch(scene.SceneURL)
						if internalId != nil {
							scene.TrailerSource = baseUrl[0] + "heresphere/videoID" + internalId[1]
						}
					case "MilfVR":
						scene.TrailerSource = "https://www.wankzvr.com/heresphere/" + id
					case "VirtualRealPassion", "VirtualRealPorn", "VirtualRealTrans", "VirtualRealGay", "VirtualRealAmateurPorn":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"script[type=\"application/ld+json\"]","content_path":"trailer.contentUrl","quality_path":"trailer.videoQuality","content_base_url":"` + baseUrl[0] + `"}`
					case "NaughtyAmerica VR":
						base := strings.Split(scene.CoverURL, "/")
						if len(base) >= 9 {
							scene.TrailerSource = `https://videos.naughtycdn.com/` + base[7] + `/trailers/vr/` + base[7] + base[8] + `/` + base[7] + base[8] + `teaser_vrdesktophd.mp4`
						}
					case "LethalHardcoreVR":
						base := strings.Split(scene.CoverURL, "/")
						if len(base) >= 5 {
							base = strings.Split(base[4], "_")
							scene.TrailerSource = `https://internal-video.adultempire.com/downloadopen/LethalVR_trailer_` + base[0] + `.mp4`
						}
					case "WhorecraftVR":
						base := strings.Split(scene.CoverURL, "/")
						if len(base) >= 5 {
							base = strings.Split(base[4], "_")
							scene.TrailerSource = `https://internal-video.adultempire.com/downloadopen/Whorecraft_trailer_` + base[0] + `.mp4`
						}
					case "CaribbeanCom VR":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"script","extract_regex":"Movie = (.+?})","content_base_url":"","record_path":"","content_path":"sample_flash_url","encoding_path":"","quality_path":""}`
					case "RealityLovers":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"script","extract_regex":"trailerUrl = \"(.+?)\";","content_base_url":"","record_path":"","content_path":"","encoding_path":"","quality_path":""}`
					case "GroobyVR":
						scene.TrailerSource = `{"scene_url":"` + scene.SceneURL + `","html_element":"dl8-video source","extract_regex":"","content_base_url":"https://www.groobyvr.com","record_path":"","content_path":"src","encoding_path":"","quality_path":"quality"}`
					}

					if scene.TrailerType != "" {
						tx.Save(&scene)
					}
				}
				return nil
			},
		},
		{
			ID: "0049-Add attributes-to-playlist-searchparams",
			Migrate: func(tx *gorm.DB) error {
				var playlists []models.Playlist

				db.Where("search_params not like '%\"attributes\":%'").Find(&playlists)
				for _, playlist := range playlists {
					playlist.SearchParams = strings.Replace(playlist.SearchParams, ",\"volume\":", ",\"attributes\":[],\"volume\":", 1)
					playlist.Save()
				}
				return nil
			},
		},
		{
			ID: "0050-remove-VirtualRealPorn-from-title",
			Migrate: func(tx *gorm.DB) error {
				var scenes []models.Scene
				err := tx.Where("title LIKE ?", "% | VirtualReal%").Find(&scenes).Error
				if err != nil {
					return err
				}

				for _, scene := range scenes {
					scene.Title = strings.TrimSpace(strings.Split(scene.Title, "|")[0])
					err = tx.Save(&scene).Error
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			ID: "0054-Update-New-Scraper-Id-in-Scene-Record",
			Migrate: func(tx *gorm.DB) error {
				var sites []models.Site
				tx.Model(&sites).Find(&sites)

				// create a scene type without deleted_at and updated_at columns
				// this means deleted records are updated as well and does not change the updated_at column
				type match struct {
					siteId         string
					sceneIdPattern string
				}
				var manualSites []match
				manualSites = append(manualSites, match{"tonightsgirlfriend", "tonight-s-girlfriend-vr%"})
				manualSites = append(manualSites, match{"littlecaprice", "little-caprice-dreams%"})
				manualSites = append(manualSites, match{"slr-originals-bts", "slr-originals-bts%"})
				manualSites = append(manualSites, match{"taboo-vr-porn", "taboo-vr-porn%"})

				var returnErr error
				for _, site := range manualSites {
					common.Log.Infof("Setting scraper_id for %s", site.siteId)
					err := tx.Model(models.Scene{}).Where("scene_id like ? and scraper_id is null", site.sceneIdPattern).Update("scraper_id", strings.ToLower(site.siteId)).Error
					if err != nil {
						returnErr = err
					}
				}
				for _, site := range sites {
					common.Log.Infof("Setting scraper_id for %s", site.Name)
					err := tx.Model(&models.Scene{}).Where("replace(scene_id,'-','') like ? and scraper_id is null", strings.Replace(site.ID, "-", "", -1)+"%").Update("scraper_id", strings.ToLower(site.ID)).Error
					if err != nil {
						returnErr = err
					}
				}

				return returnErr
			},
		},
		{
			ID: "0055-update-aggregator-scene-ids",
			Migrate: func(tx *gorm.DB) error {
				type SiteChange struct {
					SiteId    string
					NewPrefix string
				}
				//backup bundle
				common.Log.Infof("Creating pre-migration backup, please waiit, backups can take some time on a system with a large number of scenes ")
				tasks.BackupBundle(true, false, true, true, true, true, true, true, true, true, true, true, "0", "xbvr-premigration-bundle.json", "2")
				common.Log.Infof("Go to download/xbvr-premigration-bundle.json, or http://xxx.xxx.xxx.xxx:9999/download/xbvr-premigration-bundle.json if you need access to the backup")
				var sites []models.Site
				officalSiteChanges := []SiteChange{
					{"povr-originals", "povr"}, {"wankzvr", "povr"}, {"milfvr", "povr"}, {"herpovr", "povr"}, {"brasilvr", "povr"}, {"tranzvr", "povr"},
					{"slr-originals", "slr"}, {"slr-labs", "slr"}, {"slr-jav-originals", "slr"}, {"amateurcouplesvr", "slr"}, {"amateurvr3d", "slr"}, {"amorevr", "slr"}, {"astrodomina", "slr"}, {"blondehexe", "slr"}, {"blush-erotica", "slr"}, {"bravomodelsmedia", "slr"}, {"casanova", "slr"}, {"covert-japan", "slr"}, {"cuties-vr", "slr"}, {"dandy", "slr"}, {"deepinsex", "slr"}, {"deviantsvr", "slr"}, {"ellielouisevr", "slr"}, {"emilybloom", "slr"}, {"erotic-sinners", "slr"}, {"fatp", "slr"}, {"footsiebay", "slr"}, {"fuckpassvr", "slr"}, {"heathering", "slr"}, {"istripper", "slr"}, {"jackandjillvr", "slr"}, {"jimmydraws", "slr"}, {"kinkygirlsberlin", "slr"}, {"kmpvr", "slr"}, {"koalavr", "slr"}, {"lustreality", "slr"}, {"lustyvr", "slr"}, {"manny-s", "slr"}, {"mongercash", "slr"}, {"mugur-porn-vr", "slr"}, {"mutiny-vr", "slr"}, {"no2studiovr", "slr"}, {"noir", "slr"}, {"only3xvr", "slr"}, {"onlytease", "slr"}, {"peeping-thom", "slr"}, {"pervrt", "slr"}, {"petersmax", "slr"}, {"pip-vr", "slr"}, {"plushiesvr", "slr"}, {"povcentralvr", "slr"}, {"ps-porn", "slr"}, {"realhotvr", "slr"}, {"sodcreate", "slr"}, {"squeeze-vr", "slr"}, {"stockingsvr", "slr"}, {"strictlyglamourvr", "slr"}, {"stripzvr", "slr"}, {"suckmevr", "slr"}, {"swallowbay", "slr"}, {"sweetlonglips", "slr"}, {"taboo-vr-porn", "slr"}, {"tadpolexxxstudio", "slr"}, {"thatrandomeditor", "slr"}, {"tmavr", "slr"}, {"v1vr", "slr"}, {"virtualxporn", "slr"}, {"vr-pornnow", "slr"}, {"vredging", "slr"}, {"vrixxens", "slr"}, {"vrlab9division", "slr"}, {"vrmodels", "slr"}, {"vroomed", "slr"}, {"vrpfilms", "slr"}, {"vrpornjack", "slr"}, {"vrsexperts", "slr"}, {"vrsolos", "slr"}, {"vrstars", "slr"}, {"vrvids", "slr"},
					{"vrphub-vrhush", "vrphub"}, {"vrphub-stripzvr", "vrphub"},
					{"randysroadstop", "vrporn"}, {"realteensvr", "vrporn"}, {"vrclubz", "vrporn"},
				}

				isOfficalSite := func(siteList []SiteChange, siteID string) bool {
					for _, s := range siteList {
						if s.SiteId == siteID {
							return true
						}
					}
					return false
				}

				// add aggregator sites not already in officalSiteChanges
				unofficalSiteChanges := []SiteChange{{"slr-originals-bts", "slr"}}
				db.Where("name like '%)'").Find(&sites)
				for _, site := range sites {
					if !isOfficalSite(officalSiteChanges, site.ID) {
						// get (SLR), (VRPORN), etc
						re := regexp.MustCompile(`\(([^)]+)\)`)
						result := re.FindStringSubmatch(site.Name)
						newSuffix := ""
						if len(result) > 1 {
							switch result[0] {
							case "(POVR)":
								newSuffix = "povr"
							case "(SLR)":
								newSuffix = "slr"
							case "(VRP Hub)":
								newSuffix = "vrphub"
							case "(VRPorn)":
								newSuffix = "vrporn"
							default:
								common.Log.Warnf("Unknown aggregator site (%s)", site.Name)
							}
							unofficalSiteChanges = append(unofficalSiteChanges, SiteChange{site.ID, newSuffix})
						} else {
							common.Log.Warnf("Unknown aggregator site (%s)", site.Name)
						}
					}
				}

				for _, siteChange := range append(unofficalSiteChanges, officalSiteChanges...) {
					common.Log.Infof("Migrating scene_ids for %s to %s", siteChange.SiteId, siteChange.NewPrefix)
					sql := `update actions set scene_id = replace(scene_id, LOWER("` + siteChange.SiteId + `-"), "` + siteChange.NewPrefix + `-") where scene_id like "` + siteChange.SiteId + `-%"`
					tx.Exec(sql)
					sql = `update actions set scene_id = replace(replace(scene_id, '-',''), LOWER("` + siteChange.SiteId + `"), "` + siteChange.NewPrefix + `-") where scene_id not like "` + siteChange.NewPrefix + `-%" and scraper_id = "` + strings.ToLower(siteChange.SiteId) + `"`
					tx.Exec(sql)
					// set new scene_id
					sql = `update scenes set legacy_scene_id=scene_id, scene_id = replace(scene_id, LOWER("` + siteChange.SiteId + `-"), "` + siteChange.NewPrefix + `-") where scene_id like "` + siteChange.SiteId + `-%"`
					tx.Exec(sql)
					sql = `update scenes set legacy_scene_id=scene_id, scene_id = replace(replace(scene_id, '-',''), LOWER("` + siteChange.SiteId + `"), "` + siteChange.NewPrefix + `-") where  scene_id not like "` + siteChange.NewPrefix + `-%" and scraper_id = "` + strings.ToLower(siteChange.SiteId) + `"`
					tx.Exec(sql)
				}

				common.Log.Infof("Removing old sites")
				for _, site := range unofficalSiteChanges {
					// update scene scraper_id with new suffix, in case they are added back as a Custom site in scraper.json.  Needs_update to refresh new site name
					sql := fmt.Sprintf(`update scenes set scraper_id="%s", needs_update = 1 where scraper_id = "%s"`, strings.ToLower(site.SiteId+"-"+site.NewPrefix), strings.ToLower(site.SiteId))
					tx.Exec(sql)

					// delete unoffical sites from site table
					tx.Delete(&models.Site{ID: site.SiteId})
				}

				common.Log.Infof("Migrating Video Previews")
				var scenes []models.Scene
				tx.Where("legacy_scene_id is not null").Where(&models.Scene{HasVideoPreview: true}).Find(&scenes)
				for _, scene := range scenes {
					os.Rename(filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", scene.LegacySceneID)), filepath.Join(common.VideoPreviewDir, fmt.Sprintf("%v.mp4", scene.SceneID)))
				}

				common.Log.Infof("Migration needs to Reindex scenes.. please wait")
				tasks.SearchIndex()
				common.Log.Infof("Reindex of scenes complete")
				return nil
			},
		},
		{
			// Fixes the filenames of scenes for Custom SLR Sites, which have a (SLR) prefix enbedded in the filename
			ID: "0057-fix-slr-filenames-for-custom-studios",
			Migrate: func(tx *gorm.DB) error {
				common.Log.Infof("Migration updating filenames for Custom SLR Sites")
				var scenes []models.Scene
				err := tx.Where("site like ?", "% (SLR)").Find(&scenes).Error
				if err != nil {
					return err
				}

				for _, scene := range scenes {
					scene.FilenamesArr = strings.ReplaceAll(scene.FilenamesArr, "SLR_"+scene.Site, "SLR_"+strings.TrimSuffix(scene.Site, " (SLR)"))
					err = tx.Save(&scene).Error
					if err != nil {
						return err
					}
				}
				common.Log.Infof("Migration update for Custom SLR Site filenames completed")
				return nil
			},
		},
		{
			ID: "0058-add-scraper-id-to-vr-intimacy-scenes",
			Migrate: func(tx *gorm.DB) error {
				err := tx.Model(&models.Scene{}).Where("site = 'VR Intimacy'").Update("scraper_id", "czechvrintimacy").Error
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			ID: "0059-fix-czech-vr-scenes-scraper-id",
			Migrate: func(tx *gorm.DB) error {
				sites := []string{"Czech VR Casting", "Czech VR Fetish"}
				for _, site := range sites {
					scraperId := strings.ReplaceAll(strings.ToLower(site), " ", "")
					err := tx.Model(&models.Scene{}).Where("site = ? and scraper_id = 'czechvr'", site).Update("scraper_id", scraperId).Error
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			// rebuild search indexes with new fields
			ID: "0060-rebuild-new-indexes",
			Migrate: func(d *gorm.DB) error {
				os.RemoveAll(common.IndexDirV2)
				os.MkdirAll(common.IndexDirV2, os.ModePerm)
				// rebuild asynchronously, no need to hold up startup, blocking the UI
				go func() {
					tasks.SearchIndex()
					tasks.CalculateCacheSizes()
				}()
				return nil
			},
		},
		{
			// rebuild search indexes with new fields
			ID: "0061-fix-vrhush-vrallure-trailers",
			Migrate: func(tx *gorm.DB) error {
				sql := `update scenes set trailer_source = replace(trailer_source, 'deo-video source', 'web-vr-video-player source') where scraper_id in ('vrhush', 'vrallure')`
				return tx.Exec(sql).Error
			},
		},
		{
			ID: "0062-fix-vrhush-trailers",
			Migrate: func(tx *gorm.DB) error {
				sql := `update scenes set trailer_source = replace(trailer_source, '"html_element":"web-vr-video-player"', '"html_element":"web-vr-video-player source"') where scene_id like 'vrhush%' and trailer_source like '%"html_element":"web-vr-video-player"%'`
				return tx.Exec(sql).Error
			},
		},
	})

	if err := m.Migrate(); err != nil {
		common.Log.Fatalf("Could not migrate: %v", err)
	}
	common.Log.Printf("Migration did run successfully")

	db.Close()
}
