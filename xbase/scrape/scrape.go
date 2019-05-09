package scrape

import (
	"os"
	"path/filepath"

	"github.com/ProtonMail/go-appdir"
)

var appDir string
var cacheDir string

var siteCacheDir string
var sceneCacheDir string

var userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36"

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
