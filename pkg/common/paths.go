package common

import (
	"flag"
	"fmt"
	"io"
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
var DBConnectionPoolSize int
var ConcurrentScrapers int

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
	db_connection_pool_size := flag.Int("db_connection_pool_size", 0, "Optional: sets a limit to the number of db connections while scraping")
	concurrentSscrapers := flag.Int("concurrent_scrapers", 0, "Optional: sets a limit to the number of concurrent scrapers")

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
	if *db_connection_pool_size != 0 {
		DBConnectionPoolSize = *db_connection_pool_size
	} else {
		DBConnectionPoolSize = EnvConfig.DBConnectionPoolSize
	}
	if *concurrentSscrapers != 0 {
		ConcurrentScrapers = *concurrentSscrapers
	} else {
		ConcurrentScrapers = EnvConfig.ConcurrentScrapers
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

func CopyXbvrData() {
	exePath, err := os.Executable()
	if err != nil {
		Log.Warnf("Error setting up xbvr_data %s", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	sourceDir := filepath.Join(exeDir, "xbvr_data") // directory next to executable
	destDir := filepath.Join(AppDir, "xbvr_data")

	if sourceDir == destDir {
		// a normal install your xbvr executable and xbvr_data are installed seperate from your appdir
		//		if they are they same (eg a development environment) you may need to manually update your xbvr_data directory
		_, err := os.Stat(sourceDir)
		if err != nil {
			// Warning if xbvr_data doesn't exist
			Log.Warnf("Warning: data from xbvr_data is missing, setup manually if required.")
		} else {
			// Warning if xbvr_data exists, but it could be old and needs to be updated
			Log.Warnf("Not updating xbvr_data, your xbvr install location is the same as your Xbvr data directory, update xbvr_data manually if required.")
		}
		return
	}
	if err := CopyDirSkipExisting(sourceDir, destDir); err != nil {
		Log.Warnf("Error setting up xbvr_data %s", err)
		return
	}
}
func CopyDirSkipExisting(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute the relative path from src → path
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, rel)

		// If it's a directory, ensure it exists
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		destInfo, err := os.Stat(targetPath)
		if err == nil {
			// File exists → only copy if source is newer
			if !info.ModTime().After(destInfo.ModTime()) {
				return nil // skip
			}
		}

		// Copy file
		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	Log.Infof("Copying Xbvr data: %s to %s", src, dst)
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
