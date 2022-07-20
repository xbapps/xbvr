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
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
	"gopkg.in/resty.v1"
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
		return errors.New("No sites enabled!")
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
	var actions []models.Action
	db, _ := models.GetDB()
	defer db.Close()
	db.Model(&actions).
		Joins("join scenes on actions.scene_id=scenes.scene_id").
		Where("scenes.edits_applied = ?", false).
		Where("scenes.deleted_at is null").
		Find(&actions)

	for _, a := range actions {
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
	db.Model(&models.Scene{}).Update("edits_applied", true)
}

func Scrape(toScrape string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
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

			tlog.Infof("Updating tag counts")
			CountTags()
			SearchIndex()

			tlog.Infof("Reapplying edits")
			ReapplyEdits()

			tlog.Infof("Scraped %v new scenes in %s",
				sceneCount,
				time.Now().Sub(t0).Round(time.Second))
		}
	}

	models.RemoveLock("scrape")
}

func ScrapeJAVR(queryString string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
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

		tlog.Infof("Scraping R18")
		scrape.ScrapeR18(knownScenes, &collectedScenes, queryString)

		if len(collectedScenes) > 0 {
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
	models.RemoveLock("scrape")
}

func ScrapeTPDB(apiToken string, sceneUrl string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
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
	models.RemoveLock("scrape")
}

func ExportBundle() {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")
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
			fName := filepath.Join(common.AppDir, fmt.Sprintf("content-bundle-%v.json", time.Now().Unix()))
			err = ioutil.WriteFile(fName, content, 0644)
			if err == nil {
				tlog.Infof("Export completed in %v, file saved to %v", time.Now().Sub(t0), fName)
			}
		}
	}
	models.RemoveLock("scrape")
}

func ImportBundle(url string) {
	if !models.CheckLock("scrape") {
		models.CreateLock("scrape")

		tlog := log.WithField("task", "scrape")

		var bundleData ContentBundle
		tlog.Infof("Downloading bundle from URL...")
		resp, err := resty.R().SetResult(&bundleData).Get(url)

		if err == nil && resp.StatusCode() == 200 {
			db, _ := models.GetDB()
			for i := range bundleData.Scenes {
				tlog.Infof("Importing %v of %v scenes", i+1, len(bundleData.Scenes))
				models.SceneCreateUpdateFromExternal(db, bundleData.Scenes[i])
			}
			db.Close()

			tlog.Infof("Import complete")
		} else {
			tlog.Infof("Download failed!")
		}
	}
	models.RemoveLock("scrape")
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
	db, _ := models.GetDB()
	defer db.Close()

	var tags []models.Tag
	db.Model(&models.Tag{}).Find(&tags)

	type CountResults struct {
		ID          int
		Cnt         int
		Existingcnt int
	}

	var results []CountResults
	db.Model(&models.Tag{}).
		Select("tags.id, count as existingcnt, count(*) cnt").
		Group("tags.id").
		Joins("join scene_tags on scene_tags.tag_id = tags.id").
		Joins("join scenes on scenes.id=scene_tags.scene_id and scenes.deleted_at is null").
		Scan(&results)

	for i := range results {
		var tag models.Tag
		if results[i].Cnt != results[i].Existingcnt {
			db.First(&tag, results[i].ID)
			tag.Count = results[i].Cnt
			tag.Save()
		}
	}

	db.Model(&models.Actor{}).
		Select("actors.id, count as existingcnt, count(*) cnt").
		Group("actors.id").
		Joins("join scene_cast on scene_cast.actor_id = actors.id").
		Joins("join scenes on scenes.id=scene_cast.scene_id and scenes.deleted_at is null").
		Scan(&results)

	for i := range results {
		var actor models.Actor
		if results[i].Cnt != results[i].Existingcnt {
			db.First(&actor, results[i].ID)
			actor.Count = results[i].Cnt
			actor.Save()
		}
	}
	// db.Where("count = ?", 0).Delete(&Tag{})
}
