package scrape

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/ProtonMail/go-appdir"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var log = &common.Log
var appDir string
var cacheDir string

var siteCacheDir string
var sceneCacheDir string

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"

func registerScraper(id string, name string, f models.ScraperFunc) {
	models.RegisterScraper(id, name, f)
}

func logScrapeStart(id string, name string) {
	log.WithFields(logrus.Fields{
		"task":      "scraperProgress",
		"scraperID": id,
		"progress":  0,
		"started":   true,
		"completed": false,
	}).Infof("Starting %v scraper", name)
}

func logScrapeFinished(id string, name string) {
	log.WithFields(logrus.Fields{
		"task":      "scraperProgress",
		"scraperID": id,
		"progress":  0,
		"started":   false,
		"completed": true,
	}).Infof("Finished %v scraper", name)
}

func unCache(URL string, cacheDir string) {
	sum := sha1.Sum([]byte(URL))
	hash := hex.EncodeToString(sum[:])
	dir := path.Join(cacheDir, hash[:2])
	filename := path.Join(dir, hash)
	if err := os.Remove(filename); err != nil {
		log.Fatal(err)
	}
}

func updateSiteLastUpdate(id string) {
	var site models.Site
	err := site.GetIfExist(id)
	if err != nil {
		log.Error(err)
		return
	}
	site.LastUpdate = time.Now()
	site.Save()
}

func init() {
	appDir = appdir.New("xbvr").UserConfig()

	cacheDir = filepath.Join(appDir, "cache")

	siteCacheDir = filepath.Join(cacheDir, "site_cache")
	sceneCacheDir = filepath.Join(cacheDir, "scene_cache")

	_ = os.MkdirAll(appDir, os.ModePerm)
	_ = os.MkdirAll(cacheDir, os.ModePerm)

	_ = os.MkdirAll(siteCacheDir, os.ModePerm)
	_ = os.MkdirAll(sceneCacheDir, os.ModePerm)
}
