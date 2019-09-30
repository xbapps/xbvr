package xbvr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cld9x/xbvr/pkg/scrape"
	"github.com/go-test/deep"
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

func Scrape() {
	if !CheckLock("scrape") {
		CreateLock("scrape")

		tlog := log.WithField("task", "scrape")

		os.RemoveAll(filepath.Join(cacheDir, "site_cache"))

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

		tlog.Infof("Scraping BadoinkVR / 18VR / VRCosplayX / BabeVR / KinkVR")
		scrape.ScrapeBadoink(knownScenes, &collectedScenes)

		tlog.Infof("Scraping MilfVR")
		scrape.ScrapeMilfVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping NaughtyAmericaVR")
		scrape.ScrapeNA(knownScenes, &collectedScenes)

		tlog.Infof("Scraping SexBabesVR")
		scrape.ScrapeSexBabesVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VirtualRealPorn")
		scrape.ScrapeVirtualRealPorn(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VirtualTaboo")
		scrape.ScrapeVirtualTaboo(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VRBangers")
		scrape.ScrapeVRB(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VRHush")
		scrape.ScrapeVRHush(knownScenes, &collectedScenes)

		tlog.Infof("Scraping WankzVR")
		scrape.ScrapeWankz(knownScenes, &collectedScenes)

		tlog.Infof("Scraping Czech VR")
		scrape.ScrapeCzechVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping StasyQVR")
		scrape.ScrapeStasyQVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping TmwVRnet")
		scrape.ScrapeTmwVRnet(knownScenes, &collectedScenes)

		tlog.Infof("Scraping DDFNetworkVR")
		scrape.ScrapeDDFNetworkVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VRLatina")
		scrape.ScrapeVRLatina(knownScenes, &collectedScenes)

		tlog.Infof("Scraping HoloGirlsVR")
		scrape.ScrapeHoloGirlsVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping LethalHardcoreVR / WhorecraftVR")
		scrape.ScrapeLethalHardcoreVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping RealityLovers")
		scrape.ScrapeRealityLovers(knownScenes, &collectedScenes)

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
		var collectedScenes []scrape.ScrapedScene

		scrape.ScrapeBadoink(knownScenes, &collectedScenes)
		scrape.ScrapeMilfVR(knownScenes, &collectedScenes)
		scrape.ScrapeNA(knownScenes, &collectedScenes)
		scrape.ScrapeSexBabesVR(knownScenes, &collectedScenes)
		scrape.ScrapeVirtualRealPorn(knownScenes, &collectedScenes)
		scrape.ScrapeVirtualTaboo(knownScenes, &collectedScenes)
		scrape.ScrapeVRB(knownScenes, &collectedScenes)
		scrape.ScrapeVRHush(knownScenes, &collectedScenes)
		scrape.ScrapeWankz(knownScenes, &collectedScenes)
		scrape.ScrapeCzechVR(knownScenes, &collectedScenes)
		scrape.ScrapeStasyQVR(knownScenes, &collectedScenes)
		scrape.ScrapeTmwVRnet(knownScenes, &collectedScenes)
		scrape.ScrapeDDFNetworkVR(knownScenes, &collectedScenes)
		scrape.ScrapeVRLatina(knownScenes, &collectedScenes)
		scrape.ScrapeHoloGirlsVR(knownScenes, &collectedScenes)
		scrape.ScrapeLethalHardcoreVR(knownScenes, &collectedScenes)
		scrape.ScrapeRealityLovers(knownScenes, &collectedScenes)

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
