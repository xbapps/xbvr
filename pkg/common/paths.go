package common

import (
	"os"
	"path/filepath"

	"github.com/ProtonMail/go-appdir"
)

var AppDir string
var BinDir string
var CacheDir string
var ImgDir string
var IndexDirV1 string
var IndexDirV2 string
var PersistentScrapeCacheDir string
var ScrapeCacheDir string

func InitPaths() {
	AppDir = appdir.New("xbvr").UserConfig()

	CacheDir = filepath.Join(AppDir, "cache")
	BinDir = filepath.Join(AppDir, "bin")
	ImgDir = filepath.Join(AppDir, "imageproxy")
	IndexDirV1 = filepath.Join(AppDir, "search")
	IndexDirV2 = filepath.Join(AppDir, "search-v2")

	PersistentScrapeCacheDir = filepath.Join(CacheDir, "scrape_p_cache")
	ScrapeCacheDir = filepath.Join(CacheDir, "scrape_cache")

	// Remove search index v1
	if _, err := os.Stat(IndexDirV1); !os.IsNotExist(err) {
		os.RemoveAll(IndexDirV1)
	}

	_ = os.MkdirAll(AppDir, os.ModePerm)
	_ = os.MkdirAll(ImgDir, os.ModePerm)
	_ = os.MkdirAll(CacheDir, os.ModePerm)
	_ = os.MkdirAll(BinDir, os.ModePerm)
	_ = os.MkdirAll(IndexDirV2, os.ModePerm)
	_ = os.MkdirAll(PersistentScrapeCacheDir, os.ModePerm)
	_ = os.MkdirAll(ScrapeCacheDir, os.ModePerm)
}
