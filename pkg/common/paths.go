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
var MyFilesDir string
var DownloadDir string
var WebPort int

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

	enableLocalStorage := flag.Bool("localstorage", false, "Optional: Use local folder to store application data")
	app_dir := flag.String("app_dir", "", "Optional: path to the application directory")
	cache_dir := flag.String("cache_dir", "", "Optional: path to the tempoarary scraper cache directory")
	imgproxy_dir := flag.String("imgproxy_dir", "", "Optional: path to the imageproxy directory")
	search_dir := flag.String("search_dir", "", "Optional: path to the Search Index directory")
	preview_dir := flag.String("preview_dir", "", "Optional: path to the Scraper Cache directory")
	scriptsheatmap_dir := flag.String("scripts_heatmap_dir", "", "Optional: path to the scripts_heatmap directory")
	myfiles_dir := flag.String("myfiles_dir", "", "Optional: path to the myfiles directory for serving users own content (eg images")
	databaseurl := flag.String("database_url", "", "Optional: override default database path")
	web_port := flag.Int("web_port", 0, "Optional: override default Web Page port 9999")
	ws_addr := flag.String("ws_addr", "", "Optional: override default Websocket address from the default 0.0.0.0:9998")

	flag.Parse()

	if *app_dir == "" {
		tmp := os.Getenv("XBVR_APPDIR")
		app_dir = &tmp
	}
	if *app_dir == "" {
		if *enableLocalStorage {
			executable, err := os.Executable()

			if err != nil {
				panic(err)
			}

			AppDir = filepath.Dir(executable)
		} else {
			AppDir = appdir.New("xbvr").UserConfig()
		}
	} else {
		AppDir = *app_dir
	}

	CacheDir = getPath(*cache_dir, "XBVR_CACHEDIR", "cache")
	BinDir = filepath.Join(AppDir, "bin")
	ImgDir = getPath(*imgproxy_dir, "XBVR_IMAGEPROXYDIR", "imageproxy")
	MetricsDir = filepath.Join(AppDir, "metrics")
	HeatmapDir = filepath.Join(AppDir, "heatmap")
	IndexDirV2 = getPath(*search_dir, "XBVR_SEARCHDIR", "search-v2")

	ScrapeCacheDir = filepath.Join(CacheDir, "scrape_cache")

	VideoPreviewDir = getPath(*preview_dir, "XBVR_VIDEOPREVIEWDIR", "video_preview")
	VideoThumbnailDir = filepath.Join(AppDir, "video_thumbnail")
	ScriptHeatmapDir = getPath(*scriptsheatmap_dir, "XBVR_SCRIPTHEATMAPDIR", "script_heatmap")

	MyFilesDir = getPath(*myfiles_dir, "XBVR_MYFILESDIR", "myfiles")
	DownloadDir = filepath.Join(AppDir, "download")

	// Initialize DATABASE_URL once appdir path is known
	if *databaseurl != "" {
		DATABASE_URL = *databaseurl
	} else {
		if EnvConfig.DatabaseURL != "" {
			DATABASE_URL = EnvConfig.DatabaseURL
		} else {
			DATABASE_URL = fmt.Sprintf("sqlite:%v", filepath.Join(AppDir, "main.db"))
		}
	}

	if *web_port != 0 {
		WebPort = *web_port
	} else {
		WebPort = EnvConfig.WebPort
	}
	if *ws_addr != "" {
		WsAddr = *ws_addr
	} else {
		if EnvConfig.WsAddr != "" {
			WsAddr = EnvConfig.WsAddr
		}
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
	_ = os.MkdirAll(MyFilesDir, os.ModePerm)
	_ = os.MkdirAll(DownloadDir, os.ModePerm)
}
func getPath(commandLinePath string, environmentName string, directoryName string) string {
	if commandLinePath != "" {
		return commandLinePath
	}
	if os.Getenv(environmentName) != "" {
		return os.Getenv(environmentName)
	}
	return filepath.Join(AppDir, directoryName)
}
