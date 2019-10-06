package xbvr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-test/deep"
	"github.com/xbapps/xbvr/pkg/scrape"
	"gopkg.in/resty.v1"
)

type ContentBundle struct {
	Timestamp     time.Time             `json:"timestamp"`
	BundleVersion string                `json:"bundleVersion"`
	Scenes        []scrape.ScrapedScene `json:"scenes"`
}

func CleanTags() {
	RenameTags()
	CountTags()
}

func runScrapers(knownScenes []string) []scrape.ScrapedScene {
	os.RemoveAll(filepath.Join(cacheDir, "site_cache"))

	var collectedScenes []scrape.ScrapedScene

	scrapers := scrape.GetScrapers()

	var sites []Site
	db, _ := GetDB()
	db.Where(&Site{IsEnabled: true}).Find(&sites)
	db.Close()

	tlog := log.WithField("task", "scrape")

	if len(sites) > 0 {
		for _, site := range sites {
			for _, scraper := range scrapers {
				if site.ID == scraper.ID {
					tlog.Infof("Scraping %s", scraper.Name)
					scraper.Scrape(knownScenes, &collectedScenes)
					site.LastUpdate = time.Now()
					site.Save()
				}
			}
		}
	} else {
		tlog.Info("No sites enabled!")
	}

	return collectedScenes
}

func Scrape() {
	if !CheckLock("scrape") {
		CreateLock("scrape")

		tlog := log.WithField("task", "scrape")

		// Get all known scenes
		var scenes []Scene
		db, _ := GetDB()
		db.Find(&scenes)

		var knownScenes []string
		for i := range scenes {
			knownScenes = append(knownScenes, scenes[i].SceneURL)
		}

		// Start scraping
		collectedScenes := runScrapers(knownScenes)

		if len(collectedScenes) > 0 {
			tlog.Infof("Scraped %v new scenes", len(collectedScenes))

			db, _ := GetDB()
			for i := range collectedScenes {
				SceneCreateUpdateFromExternal(db, collectedScenes[i])
			}
			db.Close()

			tlog.Infof("Saved %v new scenes", len(collectedScenes))
		} else {
			tlog.Infof("No new scenes scraped")
		}
	}

	RemoveLock("scrape")
}

func ScrapeJAVR(queryString string) {
	if !CheckLock("scrape") {
		CreateLock("scrape")

		tlog := log.WithField("task", "scrape")

		// Get all known scenes
		var scenes []Scene
		db, _ := GetDB()
		db.Find(&scenes)
		db.Close()

		var knownScenes []string
		for i := range scenes {
			knownScenes = append(knownScenes, scenes[i].SceneURL)
		}

		// Start scraping
		var collectedScenes []scrape.ScrapedScene

		tlog.Infof("Scraping R18")
		scrape.ScrapeR18(knownScenes, &collectedScenes, queryString)

		if len(collectedScenes) > 0 {
			tlog.Infof("Scraped %v new scenes", len(collectedScenes))

			db, _ := GetDB()
			for i := range collectedScenes {
				SceneCreateUpdateFromExternal(db, collectedScenes[i])
			}
			db.Close()

			tlog.Infof("Saved %v new scenes", len(collectedScenes))
		} else {
			tlog.Infof("No new scenes scraped")
		}

	}
	RemoveLock("scrape")
}

func ExportBundle() {
	if !CheckLock("scrape") {
		CreateLock("scrape")

		tlog := log.WithField("task", "scrape")
		tlog.Info("Exporting content bundle...")

		var knownScenes []string
		collectedScenes := runScrapers(knownScenes)

		out := ContentBundle{
			Timestamp:     time.Now().UTC(),
			BundleVersion: "1",
			Scenes:        collectedScenes,
		}

		content, err := json.MarshalIndent(out, "", " ")
		if err == nil {
			fName := filepath.Join(appDir, fmt.Sprintf("content-bundle-%v.json", time.Now().Unix()))
			err = ioutil.WriteFile(fName, content, 0644)
			if err == nil {
				tlog.Infof("Export complete, file saved to %v", fName)
			}
		}
	}
	RemoveLock("scrape")
}

func ImportBundle(url string) {
	if !CheckLock("scrape") {
		CreateLock("scrape")

		tlog := log.WithField("task", "scrape")

		var bundleData ContentBundle
		tlog.Infof("Downloading bundle from URL...")
		resp, err := resty.R().SetResult(&bundleData).Get(url)

		if err == nil && resp.StatusCode() == 200 {
			db, _ := GetDB()
			for i := range bundleData.Scenes {
				tlog.Infof("Importing %v of %v scenes", i+1, len(bundleData.Scenes))
				SceneCreateUpdateFromExternal(db, bundleData.Scenes[i])
			}
			db.Close()

			tlog.Infof("Import complete")
		} else {
			tlog.Infof("Download failed!")
		}
	}
	RemoveLock("scrape")
}

func RenameTags() {
	db, _ := GetDB()
	defer db.Close()

	var scenes []Scene
	db.Find(&scenes)

	for i := range scenes {
		currentTags := make([]Tag, 0)
		db.Model(&scenes[i]).Related(&currentTags, "Tags")

		newTags := make([]Tag, 0)
		for j := range currentTags {
			nt := Tag{}
			if convertTag(currentTags[j].Name) != "" {
				db.Where(&Tag{Name: convertTag(currentTags[j].Name)}).FirstOrCreate(&nt)
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
	db, _ := GetDB()
	defer db.Close()

	var tags []Tag
	db.Model(&Tag{}).Find(&tags)

	for i := range tags {
		var scenes []Scene
		db.Model(tags[i]).Related(&scenes, "Scenes")

		tags[i].Count = len(scenes)
		tags[i].Save()
	}

	// db.Where("count = ?", 0).Delete(&Tag{})
}
