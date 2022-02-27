package common

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ProtonMail/go-appdir"
)

var AppDir string
var BinDir string
var CacheDir string
var ImgDir string
var MetricsDir string
var HeatmapDir string
var IndexDirV2 string
var ScrapeCacheDir string
var VideoPreviewDir string
var VideoThumbnailDir string
var ScriptHeatmapDir string

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func InitPaths() {

	enableLocalStorage := flag.Bool("localstorage", false, "Use local folder to store application data")
	flag.Parse()

	if *enableLocalStorage {
		executable, err := os.Executable()

		if err != nil {
			panic(err)
		}

		AppDir = filepath.Dir(executable)
	} else {
		AppDir = appdir.New("xbvr").UserConfig()
	}

	CacheDir = filepath.Join(AppDir, "cache")
	BinDir = filepath.Join(AppDir, "bin")
	ImgDir = filepath.Join(AppDir, "imageproxy")
	MetricsDir = filepath.Join(AppDir, "metrics")
	HeatmapDir = filepath.Join(AppDir, "heatmap")
	IndexDirV2 = filepath.Join(AppDir, "search-v2")

	ScrapeCacheDir = filepath.Join(CacheDir, "scrape_cache")

	VideoPreviewDir = filepath.Join(AppDir, "video_preview")
	VideoThumbnailDir = filepath.Join(AppDir, "video_thumbnail")
	ScriptHeatmapDir = filepath.Join(AppDir, "script_heatmap")

	// Initialize DATABASE_URL once appdir path is known
	if EnvConfig.DatabaseURL != "" {
		DATABASE_URL = EnvConfig.DatabaseURL
	} else {
		DATABASE_URL = fmt.Sprintf("sqlite:%v", filepath.Join(AppDir, "main.db"))
	}

	_ = os.MkdirAll(AppDir, os.ModePerm)
	_ = os.MkdirAll(ImgDir, os.ModePerm)
	_ = os.MkdirAll(MetricsDir, os.ModePerm)
	_ = os.MkdirAll(HeatmapDir, os.ModePerm)
	_ = os.MkdirAll(CacheDir, os.ModePerm)
	_ = os.MkdirAll(BinDir, os.ModePerm)
	_ = os.MkdirAll(IndexDirV2, os.ModePerm)
	_ = os.MkdirAll(ScrapeCacheDir, os.ModePerm)
	_ = os.MkdirAll(ScriptHeatmapDir, os.ModePerm)
}
