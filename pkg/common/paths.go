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
var MetricsDir string
var IndexDirV2 string
var PersistentScrapeCacheDir string
var ScrapeCacheDir string
var VideoPreviewDir string
var VideoThumbnailDir string

func InitPaths() {
	AppDir = appdir.New("xbvr").UserConfig()

	CacheDir = filepath.Join(AppDir, "cache")
	BinDir = filepath.Join(AppDir, "bin")
	ImgDir = filepath.Join(AppDir, "imageproxy")
	MetricsDir = filepath.Join(AppDir, "metrics")
	IndexDirV2 = filepath.Join(AppDir, "search-v2")

	PersistentScrapeCacheDir = filepath.Join(CacheDir, "scrape_p_cache")
	ScrapeCacheDir = filepath.Join(CacheDir, "scrape_cache")

	VideoPreviewDir = filepath.Join(AppDir, "video_preview")
	VideoThumbnailDir = filepath.Join(AppDir, "video_thumbnail")

	_ = os.MkdirAll(AppDir, os.ModePerm)
	_ = os.MkdirAll(ImgDir, os.ModePerm)
	_ = os.MkdirAll(MetricsDir, os.ModePerm)
	_ = os.MkdirAll(CacheDir, os.ModePerm)
	_ = os.MkdirAll(BinDir, os.ModePerm)
	_ = os.MkdirAll(IndexDirV2, os.ModePerm)
	_ = os.MkdirAll(PersistentScrapeCacheDir, os.ModePerm)
	_ = os.MkdirAll(ScrapeCacheDir, os.ModePerm)
}
