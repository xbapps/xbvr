package scrape

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/ProtonMail/go-appdir"
)

var appDir string
var cacheDir string

var siteCacheDir string
var sceneCacheDir string
var scrapers []Scraper

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"

type scraperFunc func(*sync.WaitGroup, []string, chan<- ScrapedScene) error

type Scraper struct {
	ID      string
	Name    string
	Scrape  scraperFunc
}

type ScrapedScene struct {
	SceneID     string   `json:"_id"`
	SiteID      string   `json:"scene_id"`
	SceneType   string   `json:"scene_type"`
	Title       string   `json:"title"`
	Studio      string   `json:"studio"`
	Site        string   `json:"site"`
	Covers      []string `json:"covers"`
	Gallery     []string `json:"gallery"`
	Tags        []string `json:"tags"`
	Cast        []string `json:"cast"`
	Filenames   []string `json:"filename"`
	Duration    int      `json:"duration"`
	Synopsis    string   `json:"synopsis"`
	Released    string   `json:"released"`
	HomepageURL string   `json:"homepage_url"`
}

func registerScraper(id string, name string, f scraperFunc) {
	s := Scraper{}
	s.ID = id
	s.Name = name
	s.Scrape = f
	scrapers = append(scrapers, s)
}

func GetScrapers() []Scraper {
	return scrapers
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
