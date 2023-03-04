package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-test/deep"
	"github.com/jinzhu/gorm"
	jsoniter "github.com/json-iterator/go"
	"github.com/markphelps/optional"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

type ContentBundle struct {
	Timestamp     time.Time             `json:"timestamp"`
	BundleVersion string                `json:"bundleVersion"`
	Scenes        []models.ScrapedScene `json:"scenes"`
}

type ScraperStatus struct {
	ID        string `json:"id"`
	Completed bool   `json:"completed"`
}

type BackupFileLink struct {
	SceneID string        `xbvrbackup:"scene_id"`
	Files   []models.File `xbvrbackup:"files"`
}
type BackupSceneHistory struct {
	SceneID string           `xbvrbackup:"scene_id"`
	History []models.History `xbvrbackup:"history"`
}
type BackupSceneCuepoint struct {
	SceneID   string                 `xbvrbackup:"scene_id"`
	Cuepoints []models.SceneCuepoint `xbvrbackup:"cuepoints"`
}
type BackupSceneAction struct {
	SceneID string          `xbvrbackup:"scene_id"`
	Actions []models.Action `xbvrbackup:"actions"`
}
type BackupContentBundle struct {
	Timestamp     time.Time             `xbvrbackup:"timestamp"`
	BundleVersion string                `xbvrbackup:"bundleVersion"`
	Volumne       []models.Volume       `xbvrbackup:"volumes"`
	Playlists     []models.Playlist     `xbvrbackup:"playlists"`
	Sites         []models.Site         `xbvrbackup:"sites"`
	Scenes        []models.Scene        `xbvrbackup:"scenes"`
	FilesLinks    []BackupFileLink      `xbvrbackup:"sceneFileLinks"`
	Cuepoints     []BackupSceneCuepoint `xbvrbackup:"sceneCuepoints"`
	History       []BackupSceneHistory  `xbvrbackup:"sceneHistory"`
	Actions       []BackupSceneAction   `xbvrbackup:"actions"`
	Akas          []models.Aka          `xbvrbackup:"akas"`
	TagGroups     []models.TagGroup     `xbvrbackup:"tagGroups"`
}
type RequestRestore struct {
	InclAllSites     bool   `json:"allSites"`
	OfficalSitesOnly bool   `json:"onlyIncludeOfficalSites"`
	InclScenes       bool   `json:"inclScenes"`
	InclFileLinks    bool   `json:"inclLinks"`
	InclCuepoints    bool   `json:"inclCuepoints"`
	InclHistory      bool   `json:"inclHistory"`
	InclPlaylists    bool   `json:"inclPlaylists"`
	InclActorAkas    bool   `json:"inclActorAkas"`
	InclTagGroups    bool   `json:"inclTagGroups"`
	InclVolumes      bool   `json:"inclVolumes"`
	InclSites        bool   `json:"inclSites"`
	InclActions      bool   `json:"inclActions"`
	Overwrite        bool   `json:"overwrite"`
	UploadData       string `json:"uploadData"`
}

func CleanTags() {
	RenameTags()
	CountTags()
}

func runScrapers(knownScenes []string, toScrape string, updateSite bool, collectedScenes chan<- models.ScrapedScene) error {
	defer scrape.DeleteScrapeCache()

	scrapers := models.GetScrapers()

	var sites []models.Site
	db, _ := models.GetDB()
	if toScrape == "_all" {
		db.Find(&sites)
	} else if toScrape == "_enabled" {
		db.Where(&models.Site{IsEnabled: true}).Find(&sites)
	} else {
		db.Where(&models.Site{ID: toScrape}).Find(&sites)
	}
	db.Close()

	var wg sync.WaitGroup

	if len(sites) > 0 {
		for _, site := range sites {
			for _, scraper := range scrapers {
				if site.ID == scraper.ID {
					wg.Add(1)
					go scraper.Scrape(&wg, updateSite, knownScenes, collectedScenes)
				}
			}
		}
	} else {
		return errors.New("no sites enabled")
	}

	wg.Wait()
	return nil
}

func sceneSliceAppender(collectedScenes *[]models.ScrapedScene, scenes <-chan models.ScrapedScene) {
	for scene := range scenes {
		*collectedScenes = append(*collectedScenes, scene)
	}
}

func sceneDBWriter(wg *sync.WaitGroup, i *uint64, scenes <-chan models.ScrapedScene) {
	defer wg.Done()

	db, _ := models.GetDB()
	defer db.Close()
	for scene := range scenes {
		if os.Getenv("DEBUG") != "" {
			log.Printf("Saving %v", scene.SceneID)
		}
		models.SceneCreateUpdateFromExternal(db, scene)
		atomic.AddUint64(i, 1)
		if os.Getenv("DEBUG") != "" {
			log.Printf("Saved %v", scene.SceneID)
		}
	}
}

func ReapplyEdits() {
	tlog := log.WithField("task", "scrape")

	var actions []models.Action
	db, _ := models.GetDB()
	defer db.Close()

	var count int64
	db.Model(&models.Scene{}).Where("edits_applied = ?", true).Count(&count)

	if count == 0 {
		db.Find(&actions)
	} else {
		db.Model(&actions).
			Joins("join scenes on actions.scene_id=scenes.scene_id").
			Where("scenes.edits_applied = ?", false).
			Where("scenes.deleted_at is null").
			Find(&actions)
	}

	actionCnt := 0

	for _, a := range actions {
		if actionCnt%100 == 0 {
			tlog.Infof("Processing %v of %v edits", actionCnt+1, len(actions))
		}
		actionCnt += 1

		var scene models.Scene
		err := scene.GetIfExist(a.SceneID)
		if err != nil {
			// scene has been deleted, nothing to apply
			continue
		}
		if a.ChangedColumn == "tags" || a.ChangedColumn == "cast" || a.ChangedColumn == "is_multipart" {
			prefix := string(a.NewValue[0])
			name := a.NewValue[1:]
			// Reapply Tag edits
			if a.ChangedColumn == "tags" {
				tagClean := models.ConvertTag(name)
				if tagClean != "" {
					var tag models.Tag
					db.Where(&models.Tag{Name: tagClean}).FirstOrCreate(&tag)
					if prefix == "-" {
						db.Model(&scene).Association("Tags").Delete(&tag)
					} else {
						db.Model(&scene).Association("Tags").Append(&tag)
					}
				}
			}
			// Reapply Cast edits
			if a.ChangedColumn == "cast" {
				var actor models.Actor
				db.Where(&models.Actor{Name: strings.Replace(name, ".", "", -1)}).FirstOrCreate(&actor)
				if prefix == "-" {
					db.Model(&scene).Association("Cast").Delete(&actor)
				} else {
					db.Model(&scene).Association("Cast").Append(&actor)
				}
			}
			// Reapply IsMultipart edits
			if a.ChangedColumn == "is_multipart" {
				val, _ := strconv.ParseBool(a.NewValue)
				db.Model(&scene).Update(a.ChangedColumn, val)
			}
			continue
		}
		// Reapply other edits
		db.Model(&scene).Update(a.ChangedColumn, a.NewValue)
		if a.ChangedColumn == "release_date_text" {
			dt, _ := time.Parse("2006-01-02", a.NewValue)
			db.Model(&scene).Update("release_date", dt)
		}
	}
	db.Model(&models.Scene{}).UpdateColumn("edits_applied", true)
}

func Scrape(toScrape string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")
		t0 := time.Now()
		tlog := log.WithField("task", "scrape")
		tlog.Infof("Scraping started at %s", t0.Format("Mon Jan _2 15:04:05 2006"))

		// Get all known scenes
		var scenes []models.Scene
		db, _ := models.GetDB()
		db.Find(&scenes)
		db.Close()

		var knownScenes []string
		for i := range scenes {
			if !scenes[i].NeedsUpdate {
				knownScenes = append(knownScenes, scenes[i].SceneURL)
			}
		}

		collectedScenes := make(chan models.ScrapedScene, 250)
		var sceneCount uint64

		var wg sync.WaitGroup
		wg.Add(1)
		go sceneDBWriter(&wg, &sceneCount, collectedScenes)

		// Start scraping
		if e := runScrapers(knownScenes, toScrape, true, collectedScenes); e != nil {
			tlog.Info(e)
		} else {
			// Notify DB Writer threads that there are no more scenes
			close(collectedScenes)

			// Wait for DB Writer threads to complete
			wg.Wait()

			// Send a signal to clean up the progress bars just in case
			log.WithField("task", "scraperProgress").Info("DONE")

			var dummyAka models.Aka
			dummyAka.UpdateAkaSceneCastRecords()

			var dummyTagGroup models.TagGroup
			dummyTagGroup.UpdateSceneTagRecords()

			tlog.Infof("Updating tag counts")
			CountTags()
			dummyAka.RefreshAkaActorNames()
			SearchIndex()

			tlog.Infof("Reapplying edits")
			ReapplyEdits()

			tlog.Infof("Scraped %v new scenes in %s",
				sceneCount,
				time.Now().Sub(t0).Round(time.Second))
		}
	}
}

func ScrapeJAVR(queryString string, scraper string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")
		t0 := time.Now()
		tlog := log.WithField("task", "scrape")
		tlog.Infof("Scraping started at %s", t0.Format("Mon Jan _2 15:04:05 2006"))

		// Start scraping
		var collectedScenes []models.ScrapedScene

		if scraper == "javlibrary" {
			tlog.Infof("Scraping JavLibrary")
			scrape.ScrapeJavLibrary(&collectedScenes, queryString)
		} else if scraper == "javbus" {
			tlog.Infof("Scraping JavBus")
			scrape.ScrapeJavBus(&collectedScenes, queryString)
		} else if scraper == "javland" {
			tlog.Infof("Scraping JavLand")
			scrape.ScrapeJavLand(&collectedScenes, queryString)
		} else {
			tlog.Infof("Scraping JavDB")
			scrape.ScrapeJavDB(&collectedScenes, queryString)
		}

		if len(collectedScenes) > 0 {
			db, _ := models.GetDB()
			for i := range collectedScenes {
				models.SceneCreateUpdateFromExternal(db, collectedScenes[i])
			}
			db.Close()

			tlog.Infof("Updating tag counts")
			CountTags()
			IndexScrapedScenes(&collectedScenes)

			tlog.Infof("Scraped %v new scenes in %s",
				len(collectedScenes),
				time.Now().Sub(t0).Round(time.Second))
		} else {
			tlog.Infof("No new scenes scraped")
		}

	}
}

func ScrapeTPDB(apiToken string, sceneUrl string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")
		t0 := time.Now()
		tlog := log.WithField("task", "scrape")
		tlog.Infof("Scraping started at %s", t0.Format("Mon Jan _2 15:04:05 2006"))

		// Get all known scenes
		var scenes []models.Scene
		db, _ := models.GetDB()
		db.Find(&scenes)
		db.Close()

		var knownScenes []string
		for i := range scenes {
			knownScenes = append(knownScenes, scenes[i].SceneURL)
		}

		// Start scraping
		var collectedScenes []models.ScrapedScene

		tlog.Infof("Scraping TPDB")
		err := scrape.ScrapeTPDB(knownScenes, &collectedScenes, apiToken, sceneUrl)

		if err != nil {
			tlog.Errorf(err.Error())
		} else if len(collectedScenes) > 0 {
			// At this point we know the API Token is correct, so we will save
			// it to the config store
			if config.Config.Vendor.TPDB.ApiToken != apiToken {
				config.Config.Vendor.TPDB.ApiToken = apiToken
				config.SaveConfig()
			}

			db, _ := models.GetDB()
			for i := range collectedScenes {
				models.SceneCreateUpdateFromExternal(db, collectedScenes[i])
			}
			db.Close()

			tlog.Infof("Updating tag counts")
			CountTags()
			SearchIndex()

			tlog.Infof("Scraped %v new scenes in %s",
				len(collectedScenes),
				time.Now().Sub(t0).Round(time.Second))
		} else {
			tlog.Infof("No new scenes scraped")
		}

	}
}

func ExportBundle() {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")
		t0 := time.Now()

		tlog := log.WithField("task", "scrape")
		tlog.Info("Exporting content bundle...")

		var knownScenes []string
		collectedScenes := make(chan models.ScrapedScene, 100)

		var scrapedScenes []models.ScrapedScene
		go sceneSliceAppender(&scrapedScenes, collectedScenes)

		runScrapers(knownScenes, "_enabled", false, collectedScenes)

		out := ContentBundle{
			Timestamp:     time.Now().UTC(),
			BundleVersion: "1",
			Scenes:        scrapedScenes,
		}

		content, err := json.MarshalIndent(out, "", " ")
		if err == nil {
			fName := filepath.Join(common.DownloadDir, fmt.Sprintf("content-bundle-v1-%v.json", time.Now().Unix()))
			err = ioutil.WriteFile(fName, content, 0644)
			if err == nil {
				tlog.Infof("Export completed in %v, file saved to %v", time.Now().Sub(t0), fName)
			}
		}
	}
}

func ImportBundle(uploadData string) {

	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")

		tlog := log.WithField("task", "scrape")

		var bundleData ContentBundle
		tlog.Infof("Restoring bundle ...")
		var err error

		json.Unmarshal([]byte(uploadData), &bundleData)

		if err == nil {
			if bundleData.BundleVersion != "1" {
				tlog.Infof("Restore Failed! Bundle file is version %v, version 1 expected", bundleData.BundleVersion)
				return
			}

			ImportBundleV1(bundleData)
			tlog.Infof("Import complete")
		} else {
			tlog.Infof("Download failed!")
		}
	}
}

func ImportBundleV1(bundleData ContentBundle) {
	tlog := log.WithField("task", "scrape")
	db, _ := models.GetDB()
	defer db.Close()

	for i := range bundleData.Scenes {
		tlog.Infof("Importing %v of %v scenes", i+1, len(bundleData.Scenes))
		models.SceneCreateUpdateFromExternal(db, bundleData.Scenes[i])
	}

}

func BackupBundle(inclAllSites bool, onlyIncludeOfficalSites bool, inclScenes bool, inclFileLinks bool, inclCuepoints bool, inclHistory bool, inclPlaylists bool, InclActorAkas bool, inclTagGroups bool, inclVolumes bool, inclSites bool, inclActions bool, playlistId string, outputBundleFilename string, version string) string {
	var out BackupContentBundle
	var content []byte
	exportCnt := 0

	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")
		t0 := time.Now()

		tlog := log.WithField("task", "scrape")
		tlog.Info("Backing up content bundle...")

		if outputBundleFilename == "" {
			outputBundleFilename = "xbvr-content-bundle.json"
		}
		if version == "" {
			version = "2.1"
		}

		db, _ := models.GetDB()
		defer db.Close()

		var scenes []models.Scene
		backupSceneList := []models.Scene{}
		backupCupointList := []BackupSceneCuepoint{}
		backupFileLinkList := []BackupFileLink{}
		backupHistoryList := []BackupSceneHistory{}
		backupActionList := []BackupSceneAction{}

		if inclScenes || inclFileLinks || inclCuepoints || inclHistory || inclActions {
			var selectedSites []models.Site
			if !inclAllSites || onlyIncludeOfficalSites {
				tx := db.Model(&selectedSites)
				if !inclAllSites {
					tx = tx.Where(&models.Site{IsEnabled: true})
				}
				if onlyIncludeOfficalSites {
					tx = tx.Where("name not like ?", "%(Custom %)")
				}
				tx.Find(&selectedSites)
			}

			if playlistId != "0" {
				// the user selected a Saved Search, filter scenes on that
				playlist := models.Playlist{}
				db.First(&playlist, playlistId)
				var r models.RequestSceneList
				json.Unmarshal([]byte(playlist.SearchParams), &r)
				r.Limit = optional.NewInt(100000)

				q := models.QueryScenes(r, false)
				scenes = q.Scenes
			} else {
				// no saved search, so get all scenes
				db.Select("id, scene_id").Find(&scenes)
			}

			var err error
			for cnt, scene := range scenes {
				if cnt%500 == 0 {
					tlog.Infof("Reading scene %v of %v, selected %v scenes", cnt+1, len(scenes), exportCnt)
				}

				// check if the scene is for a site we want
				if !inclAllSites || onlyIncludeOfficalSites {
					idx := FindSite(selectedSites, GetScraperId(scene.SceneID, db))
					if idx < 0 {
						continue
					}
				}

				err = db.Preload("Files").
					Preload("Cuepoints").
					Preload("History").
					// do not export tag groups  or they will load back as real tags not tag groups
					Preload("Tags", "substr(name, 1, 10)<>'tag group:'").
					// do not export aka actors or they will load back as real actors not aka groups
					Preload("Cast", "substr(name, 1, 4)<>'aka:'").
					Where(&models.Scene{ID: scene.ID}).First(&scene).Error

				if err != nil {
					tlog.Errorf("Error reading scene %s", scene.SceneID)
				}

				if len(scene.History) > 0 && inclHistory {
					backupHistoryList = append(backupHistoryList, BackupSceneHistory{SceneID: scene.SceneID, History: scene.History})
				}

				sceneAction := []models.Action{}
				if inclActions {
					db.Where(&models.Action{SceneID: scene.SceneID}).Find(&sceneAction)
					if len(sceneAction) > 0 {
						backupActionList = append(backupActionList, BackupSceneAction{SceneID: scene.SceneID, Actions: sceneAction})
					}
				}

				if inclCuepoints && len(scene.Cuepoints) > 0 {
					backupCupointList = append(backupCupointList, BackupSceneCuepoint{SceneID: scene.SceneID, Cuepoints: scene.Cuepoints})
				}
				if inclFileLinks && len(scene.Files) > 0 {
					backupFileLinkList = append(backupFileLinkList, BackupFileLink{SceneID: scene.SceneID, Files: scene.Files})
				}
				if inclScenes {
					scene.Files = []models.File{}
					scene.Cuepoints = []models.SceneCuepoint{}
					scene.History = []models.History{}
					backupSceneList = append(backupSceneList, scene)
				}
				if err != nil {
					tlog.Errorf("Error reading scene Id %v of %s", scene.ID, err)
				}
				exportCnt += 1
			}
		}

		var volumes []models.Volume
		if inclVolumes {
			db.Find(&volumes)
		}
		var playlists []models.Playlist
		if inclPlaylists {
			db.Find(&playlists)
		}

		var sites []models.Site
		if inclSites {
			db.Find(&sites)
		}

		var akas []models.Aka
		if InclActorAkas {
			db.Preload("AkaActor").Preload("Akas").Find(&akas)
		}

		var tagGroups []models.TagGroup
		if inclTagGroups {
			db.Preload("TagGroupTag").Preload("Tags").Find(&tagGroups)
		}

		var err error
		out = BackupContentBundle{
			Timestamp:     time.Now().UTC(),
			BundleVersion: version,
			Volumne:       volumes,
			Playlists:     playlists,
			Sites:         sites,
			Scenes:        backupSceneList,
			FilesLinks:    backupFileLinkList,
			Cuepoints:     backupCupointList,
			History:       backupHistoryList,
			Actions:       backupActionList,
			Akas:          akas,
			TagGroups:     tagGroups,
		}

		var json = jsoniter.Config{
			EscapeHTML:             true,
			SortMapKeys:            true,
			ValidateJsonRawMessage: true,
			TagKey:                 "xbvrbackup",
		}.Froze()
		content, err = json.MarshalIndent(out, "", " ")

		if err == nil {
			fName := filepath.Join(common.DownloadDir, outputBundleFilename)
			err = ioutil.WriteFile(fName, content, 0644)
			if err == nil {
				tlog.Infof("Backup file generated in %v, %v scenes selected, ready to download", time.Since(t0), exportCnt)
			} else {
				tlog.Infof("Error in Backup file generation %v, %v scenes selected, ready to download", time.Since(t0), exportCnt)
			}
		}
	}
	return string(content)
}

func RestoreBundle(request RequestRestore) {
	if strings.Contains(request.UploadData, "\"bundleVersion\":\"1\"") {
		ImportBundle(request.UploadData)
		return
	}
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
		defer models.RemoveLock("scrape")

		tlog := log.WithField("task", "scrape")

		var json = jsoniter.Config{
			EscapeHTML:             true,
			SortMapKeys:            true,
			ValidateJsonRawMessage: true,
			TagKey:                 "xbvrbackup",
		}.Froze()

		var bundleData BackupContentBundle
		var err error
		tlog.Infof("Restoring data ...")

		json.UnmarshalFromString(request.UploadData, &bundleData)

		if err == nil {
			if bundleData.BundleVersion != "2.1" {
				tlog.Infof("Restore Failed! Bundle file is version %v, version %v expected", bundleData.BundleVersion, "2.1")
				return
			}
			db, _ := models.GetDB()
			defer db.Close()

			var selectedSites []models.Site
			if !request.InclAllSites || request.OfficalSitesOnly {
				tx := db.Model(&selectedSites)
				if !request.InclAllSites {
					tx = tx.Where(&models.Site{IsEnabled: true})
				}
				if request.OfficalSitesOnly {
					tx = tx.Where("name not like ?", "%(Custom %)")
				}
				tx.Find(&selectedSites)
			}

			if request.InclVolumes {
				RestoreMediaPaths(bundleData.Volumne, request.Overwrite, db)
			}
			if request.InclPlaylists {
				RestorePlaylist(bundleData.Playlists, request.Overwrite, db)
			}
			if request.InclSites {
				RestoreSites(bundleData.Sites, request.Overwrite, db)
			}
			if request.InclScenes {
				RestoreScenes(bundleData.Scenes, request.InclAllSites, selectedSites, request.Overwrite, request.InclCuepoints, request.InclFileLinks, request.InclHistory, db)
			}
			if request.InclCuepoints {
				RestoreCuepoints(bundleData.Cuepoints, request.InclAllSites, selectedSites, request.Overwrite, db)
			}
			if request.InclFileLinks {
				RestoreSceneFileLinks(bundleData.FilesLinks, request.InclAllSites, selectedSites, request.Overwrite, db)
			}
			if request.InclHistory {
				RestoreHistory(bundleData.History, request.InclAllSites, selectedSites, request.Overwrite, db)
			}
			if request.InclActions {
				RestoreActions(bundleData.Actions, request.InclAllSites, selectedSites, request.Overwrite, db)
			}
			if request.InclActorAkas {
				RestoreAkas(bundleData.Akas, request.Overwrite, db)
			}
			if request.InclTagGroups {
				RestoreTagGroups(bundleData.TagGroups, request.Overwrite, db)
			}

			if request.InclScenes || request.InclFileLinks {
				UpdateSceneStatus(db)
			}
			if request.InclScenes {
				CountTags()
				IndexScenes(&(bundleData.Scenes))
			}

			if request.InclScenes || request.InclActorAkas {
				var aka models.Aka
				aka.UpdateAkaSceneCastRecords()
			}
			if request.InclScenes || request.InclTagGroups {
				var tagGroup models.TagGroup
				tagGroup.UpdateSceneTagRecords()
			}

			tlog.Infof("Restore complete")
		} else {
			tlog.Infof("Restore failed!")
		}
	}
}

func RestoreScenes(scenes []models.Scene, inclAllSites bool, selectedSites []models.Site, overwrite bool, inclCuepoints bool, inclFileLinks bool, inclHistory bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring scenes")

	addedCnt := 0
	for sceneCnt, scene := range scenes {
		if sceneCnt%250 == 0 {
			tlog.Infof("Processing %v of %v scenes", sceneCnt+1, len(scenes))
		}
		// check if the scene is for a site we want
		if !inclAllSites {
			idx := FindSite(selectedSites, scene.ScraperId)
			if idx < 0 {
				continue
			}
		}
		var found models.Scene
		db.Where(&models.Scene{SceneID: scene.SceneID}).First(&found)

		for i := 0; i <= len(scene.Cast)-1; i++ {
			var tmpActor models.Actor
			db.Where(&models.Actor{Name: scene.Cast[i].Name}).FirstOrCreate(&tmpActor)
			scene.Cast[i] = tmpActor
		}
		for i := 0; i <= len(scene.Tags)-1; i++ {
			var tmpTag models.Tag
			db.Where(&models.Tag{Name: scene.Tags[i].Name}).FirstOrCreate(&tmpTag)
			scene.Tags[i] = tmpTag
		}
		var site models.Site
		siteErr := site.GetIfExist(scene.ScraperId)
		if siteErr != nil {
			scene.IsSubscribed = site.Subscribed
		}

		if found.ID == 0 { // id = 0 is a new record
			scene.ID = 0 // dont use the id from json
			models.SaveWithRetry(db, &scene)
			addedCnt++
		} else {
			if overwrite {
				scene.ID = found.ID // use the Id from the existing db record
				models.SaveWithRetry(db, &scene)
				addedCnt++
			}
		}
	}
	tlog.Infof("%v Scenes restored", addedCnt)
}

func RestoreCuepoints(sceneCuepointList []BackupSceneCuepoint, inclAllSites bool, selectedSites []models.Site, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring scene cuepoints")

	addedCnt := 0
	for cnt, cuepoints := range sceneCuepointList {
		if cnt%500 == 0 {
			tlog.Infof("Processing cuepoints %v of %v", cnt+1, len(sceneCuepointList))
		}
		// check if the scene is for a site we want
		if !inclAllSites {
			idx := FindSite(selectedSites, GetScraperId(cuepoints.SceneID, db))
			if idx < 0 {
				continue
			}
		}
		var found models.Scene
		db.Preload("Cuepoints").Where(&models.Scene{SceneID: cuepoints.SceneID}).First(&found)
		if found.ID == 0 || len(cuepoints.Cuepoints)+len(found.Cuepoints) == 0 {
			continue
		} else {
			for i, cp := range cuepoints.Cuepoints {
				cp.SceneID = found.ID
				cp.ID = 0
				cuepoints.Cuepoints[i] = cp
			}
			if overwrite || len(found.Cuepoints) == 0 {
				if len(cuepoints.Cuepoints)+len(found.Cuepoints) > 0 {
					if len(found.Cuepoints) > 0 {
						err := db.Delete(&models.SceneCuepoint{}, "scene_id = ?", found.ID).Error
						//models.SaveWithRetry(db, &del)
						if err != nil {
							tlog.Infof("Eror deleteing cuepoints")
						}
					}
					found.Cuepoints = cuepoints.Cuepoints
					models.SaveWithRetry(db, &found)
					addedCnt++
				}
			}
		}
	}
	tlog.Infof("%v Scenes with cuepoints restored", addedCnt)
}

func RestoreSceneFileLinks(backupFileList []BackupFileLink, inclAllSites bool, selectedSites []models.Site, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring scene matched files")

	var volumes []models.Volume
	db.Find(&volumes)

	addedCnt := 0
	for cnt, backupSceneFiles := range backupFileList {
		if cnt%500 == 0 {
			tlog.Infof("Processing files %v of %v", cnt+1, len(backupFileList))
		}

		// check if the scene is for a site we want
		if !inclAllSites {
			idx := FindSite(selectedSites, GetScraperId(backupSceneFiles.SceneID, db))
			if idx < 0 {
				continue
			}
		}
		addedCnt++
		if overwrite {
			db.Delete(&models.File{}, "scene_id = ?", backupSceneFiles.SceneID)
		}

		for _, scenefile := range backupSceneFiles.Files {
			var found models.File
			db.Where(&models.File{Filename: scenefile.Filename, Path: scenefile.Path}).Find(&found)

			if found.ID == 0 {
				scenefile.ID = 0
				voldId := FindNewVolumeId(volumes, scenefile.Path)
				if voldId == -1 {
					tlog.Infof("No volume for path %s, skipping", scenefile.Path)
					continue // no volume, can't add
				}
				s := models.Scene{}
				db.Where(&models.Scene{SceneID: backupSceneFiles.SceneID}).Find(&s)
				scenefile.SceneID = s.ID
				scenefile.VolumeID = uint(voldId)
				models.SaveWithRetry(db, &scenefile)
			} else {
				if found.SceneID == 0 {
					s := models.Scene{}
					db.Where(&models.Scene{SceneID: backupSceneFiles.SceneID}).Find(&s)
					found.SceneID = s.ID
					models.SaveWithRetry(db, &found)
				}
			}
		}
	}
	tlog.Infof("%v Scenes with file links restored", addedCnt)
}

func RestoreHistory(sceneHistoryList []BackupSceneHistory, inclAllSites bool, selectedSites []models.Site, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring scene watch history")

	addedCnt := 0
	for cnt, histories := range sceneHistoryList {
		if cnt%500 == 0 {
			tlog.Infof("Processing history %v of %v", cnt+1, len(sceneHistoryList))
		}
		// check if the scene is for a site we want
		if !inclAllSites {
			idx := FindSite(selectedSites, GetScraperId(histories.SceneID, db))
			if idx < 0 {
				continue
			}
		}
		var found models.Scene
		db.Preload("History").Where(&models.Scene{SceneID: histories.SceneID}).First(&found)
		if found.ID == 0 || len(histories.History)+len(found.History) == 0 {
			continue
		} else {
			changed := false
			for i, cp := range histories.History {
				cp.SceneID = found.ID
				cp.ID = 0
				histories.History[i] = cp
			}
			if overwrite || len(found.History) == 0 {
				if len(histories.History)+len(found.History) > 0 {
					if len(found.History) > 0 {
						err := db.Delete(&models.History{}, "scene_id = ?", found.ID).Error
						//models.SaveWithRetry(db, &del)
						if err != nil {
							tlog.Infof("Eror deleteing history")
						}
					}
					found.History = histories.History
					models.SaveWithRetry(db, &found)
					addedCnt++
				}
			} else {
				for _, historyEntry := range histories.History {
					cpIdx, _ := CheckHistory(found.History, historyEntry)
					if cpIdx < 0 {
						found.History = append(found.History, historyEntry)
						changed = true
					}
				}
				if changed {
					models.SaveWithRetry(db, &found)
					addedCnt++
				}
			}
		}
	}
	tlog.Infof("%v Scenes with history restored", addedCnt)
}
func RestoreActions(sceneActionList []BackupSceneAction, inclAllSites bool, selectedSites []models.Site, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring scene edits")

	addedCnt := 0
	for cnt, actions := range sceneActionList {
		if cnt%500 == 0 {
			tlog.Infof("Processing actions %v of %v", cnt+1, len(sceneActionList))
		}
		// check if the scene is for a site we want
		if !inclAllSites {
			idx := FindSite(selectedSites, GetScraperId(actions.SceneID, db))
			if idx < 0 {
				continue
			}
		}

		if overwrite {
			if len(actions.Actions) > 0 {
				err := db.Delete(&models.History{}, "scene_id = ?", actions.SceneID).Error
				if err != nil {
					tlog.Infof("Eror deleteing history")
				}
			}
		} else {
			var existingAction models.Action
			db.Where(&models.Action{SceneID: actions.SceneID}).First(&existingAction)
			if existingAction.ID > 0 {
				tlog.Infof("Actions already exist for scene %s, cannot add new actions, use Overwrite+New", actions.SceneID)
				continue
			}

		}
		for _, action := range actions.Actions {
			action.ID = 0
			models.SaveWithRetry(db, &action)
		}
		addedCnt++
	}
	tlog.Infof("%v Scenes with actions restored", addedCnt)
}

func RestoreMediaPaths(mediaPaths []models.Volume, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring media paths")

	addedCnt := 0
	for _, mediaPath := range mediaPaths {
		var found models.Volume
		db.Where(&models.Volume{Path: mediaPath.Path}).First(&found)

		if found.ID == 0 { // id = 0 is a new record
			mediaPath.ID = 0 // dont use the id from json
			models.SaveWithRetry(db, &mediaPath)
			addedCnt++
		} else {
			if overwrite {
				mediaPath.ID = found.ID // use the Id from the existing db record
				models.SaveWithRetry(db, &mediaPath)
				addedCnt++
			}
		}
	}
	tlog.Infof("%v Media paths restored", addedCnt)
}

func RestorePlaylist(playlists []models.Playlist, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring playlists")

	addedCnt := 0
	for _, playlist := range playlists {
		var found models.Playlist
		db.Where(&models.Playlist{Name: playlist.Name}).First(&found)

		if found.ID == 0 { // id = 0 is a new record
			playlist.ID = 0 // dont use the id from json
			models.SaveWithRetry(db, &playlist)
			addedCnt++
		} else {
			if overwrite {
				playlist.ID = found.ID // use the Id from the existing db record
				models.SaveWithRetry(db, &playlist)
				addedCnt++
			}
		}
	}
	tlog.Infof("%v Saved Searches restored", addedCnt)
}

func RestoreSites(sites []models.Site, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring sites")

	addedCnt := 0
	for _, site := range sites {
		var found models.Site
		db.Where(&models.Site{Name: site.Name}).First(&found)

		if found.ID != "" { // id = "" is a new record
			// restore fields that should not be overwritten from the eisting record
			site.ID = found.ID
			site.AvatarURL = found.AvatarURL
			site.IsBuiltin = found.IsBuiltin
			site.LastUpdate = found.LastUpdate
			models.SaveWithRetry(db, &site)
			addedCnt++
		}
		db.Model(&models.Scene{}).Where("scraper_id = ?", site.ID).Update("is_subscribed", site.Subscribed)
	}
	tlog.Infof("%v Sites  restored", addedCnt)
}

func RestoreAkas(akas []models.Aka, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring Actor Akas")

	addedCnt := 0
	for _, aka := range akas {
		var found models.Aka
		name := aka.AkaNameSortedAlphabetcally()
		db.Where(&models.Aka{Name: name}).Preload("AkaActor").First(&found)

		if found.ID == 0 { // id = 0 is a new record
			CheckActors(&aka, 0, db)
			aka.ID = 0 // dont use the id from json
			aka.Name = name
			models.SaveWithRetry(db, &aka)
			addedCnt++
		} else {
			if overwrite {
				CheckActors(&aka, found.AkaActorId, db)
				aka.ID = found.ID // use the Id from the existing db record
				models.SaveWithRetry(db, &aka)
				addedCnt++
			}
		}
	}
	tlog.Infof("%v Actor Akas restored", addedCnt)
}

func CheckActors(aka *models.Aka, aka_actor_id uint, db *gorm.DB) {
	// check an aka actor exists
	if aka_actor_id == 0 {
		models.SaveWithRetry(db, &aka.AkaActor)
		aka.AkaActorId = aka.AkaActor.ID
	} else {
		aka.AkaActorId = aka_actor_id
		aka.AkaActor.ID = aka_actor_id
	}
	for idx, actor := range aka.Akas {
		var found models.Actor

		db.Where(&models.Actor{Name: actor.Name}).First(&found)
		if found.ID != 0 {
			//models.SaveWithRetry(db, &found)
			aka.Akas[idx].ID = found.ID
		}
	}

}

func RestoreTagGroups(tagGroups []models.TagGroup, overwrite bool, db *gorm.DB) {
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Restoring Tag Groups")

	addedCnt := 0
	for _, tagGroup := range tagGroups {
		var found models.TagGroup
		db.Where(&models.TagGroup{Name: tagGroup.Name}).Preload("TagGrou").First(&found)

		if found.ID == 0 { // id = 0 is a new record
			CheckTagGroup(&tagGroup, 0, db)
			tagGroup.ID = 0 // dont use the id from json
			models.SaveWithRetry(db, &tagGroup)
			addedCnt++
		} else {
			if overwrite {
				CheckTagGroup(&tagGroup, found.TagGroupTagId, db)
				tagGroup.ID = found.ID // use the Id from the existing db record
				models.SaveWithRetry(db, &tagGroup)
				addedCnt++
			}
		}
	}
	tlog.Infof("%v Tag Groups restored", addedCnt)
}

func CheckTagGroup(tagGroup *models.TagGroup, tag_group_tag_id uint, db *gorm.DB) {
	// check an tag grouup exists
	if tag_group_tag_id == 0 {
		models.SaveWithRetry(db, &tagGroup.TagGroupTag)
		tagGroup.TagGroupTagId = tagGroup.TagGroupTag.ID
	} else {
		tagGroup.TagGroupTagId = tag_group_tag_id
		tagGroup.TagGroupTag.ID = tag_group_tag_id
	}
	for idx, tag := range tagGroup.Tags {
		var found models.Tag

		db.Where(&models.Tag{Name: tag.Name}).First(&found)
		if found.ID != 0 {
			tagGroup.Tags[idx].ID = found.ID
		}
	}
}

func RenameTags() {
	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Find(&scenes)

	for i := range scenes {
		currentTags := make([]models.Tag, 0)
		db.Model(&scenes[i]).Related(&currentTags, "Tags")

		newTags := make([]models.Tag, 0)
		for j := range currentTags {
			nt := models.Tag{}
			if models.ConvertTag(currentTags[j].Name) != "" {
				db.Where(&models.Tag{Name: models.ConvertTag(currentTags[j].Name)}).FirstOrCreate(&nt)
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
	var tag models.Tag
	tag.CountTags()

	var actor models.Actor
	actor.CountActorTags()
}

func FindSite(sites []models.Site, scraperId string) int {
	for i, site := range sites {
		if scraperId == site.ID {
			return i
		}
	}
	return -1
}

func GetScraperId(sceneId string, db *gorm.DB) string {
	var scene models.Scene
	db.Where(models.Scene{SceneID: sceneId}).First(&scene)
	return scene.ScraperId
}

func CheckCuepoint(cuepoints []models.SceneCuepoint, findCuepoint models.SceneCuepoint) (int, bool) {
	for i, cuepoint := range cuepoints {
		if cuepoint.TimeStart == findCuepoint.TimeStart {
			if cuepoint.Name == findCuepoint.Name {
				return i, true
			} else {
				return i, false
			}
		}
	}
	return -1, false
}
func CheckFiles(files []models.File, findFiles models.File) (int, bool) {
	for i, file := range files {
		if file.Filename == findFiles.Filename && file.Path == findFiles.Path {
			return i, true
		}
	}
	return -1, false
}
func CheckHistory(historyList []models.History, findHistory models.History) (int, bool) {
	for i, historyEntry := range historyList {
		if historyEntry.TimeStart == findHistory.TimeStart {
			return i, true
		}
	}
	return -1, false
}
func FindNewVolumeId(volumes []models.Volume, path string) int {
	for _, vol := range volumes {
		if strings.HasPrefix(path, vol.Path) {
			return int(vol.ID)
		}
	}
	return -1
}

func UpdateSceneStatus(db *gorm.DB) {
	// Update scene statuses
	tlog := log.WithField("task", "scrape")
	tlog.Infof("Update status of Scenes")
	scenes := []models.Scene{}
	db.Model(&models.Scene{}).Find(&scenes)

	for i := range scenes {
		scenes[i].UpdateStatus()
		if (i % 70) == 0 {
			tlog.Infof("Update status of Scenes (%v/%v)", i+1, len(scenes))
		}
	}
}
