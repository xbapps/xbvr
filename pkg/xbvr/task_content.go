package xbvr

import (
	"time"

	"github.com/cld9x/xbvr/pkg/scrape"
	"github.com/go-test/deep"
	"gopkg.in/resty.v1"
)

type Bundle struct {
	Timestamp     time.Time             `json:"timestamp"`
	BundleVersion string                `json:"bundle_version"`
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

		tlog.Infof("Scraping NaughtyAmericaVR")
		scrape.ScrapeNA(knownScenes, &collectedScenes)

		tlog.Infof("Scraping BadoinkVR / 18VR / VRCosplayX / BabeVR / KinkVR")
		scrape.ScrapeBadoink(knownScenes, &collectedScenes)

		tlog.Infof("Scraping MilfVR")
		scrape.ScrapeMilfVR(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VRBangers")
		scrape.ScrapeVRB(knownScenes, &collectedScenes)

		tlog.Infof("Scraping WankzVR")
		scrape.ScrapeWankz(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VirtualTaboo")
		scrape.ScrapeVirtualTaboo(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VirtualRealPorn")
		scrape.ScrapeVirtualRealPorn(knownScenes, &collectedScenes)

		tlog.Infof("Scraping VRHush")
		scrape.ScrapeVRHush(knownScenes, &collectedScenes)

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

func ImportBundle() {
	if !CheckLock("scrape") {
		CreateLock("scrape")

		tlog := log.WithField("task", "scrape")

		var bundleData Bundle
		resp, err := resty.R().SetResult(&bundleData).Get("http://127.0.0.1:9999/static/bundle.json")

		tlog.Info(err)

		if err == nil && resp.StatusCode() == 200 {
			db, _ := GetDB()
			for i := range bundleData.Scenes {
				tlog.Infof("Importing %v of %v scenes", i+1, len(bundleData.Scenes))
				SceneCreateUpdateFromExternal(db, bundleData.Scenes[i])
			}
			db.Close()
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
