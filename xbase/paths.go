package xbase

import (
	"os"
	"path/filepath"

	"github.com/ProtonMail/go-appdir"
)

var appDir string

var imgDir string
var cacheDir string

func initPaths() {
	appDir = appdir.New("xbvr").UserConfig()

	imgDir = filepath.Join(appDir, "imageproxy")
	cacheDir = filepath.Join(appDir, "cache")

	_ = os.MkdirAll(appDir, os.ModePerm)
	_ = os.MkdirAll(imgDir, os.ModePerm)
	_ = os.MkdirAll(cacheDir, os.ModePerm)
}