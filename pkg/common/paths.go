package common

import (
	"os"
	"path/filepath"

	"github.com/ProtonMail/go-appdir"
)

var AppDir string

var ImgDir string
var CacheDir string
var BinDir string
var IndexDir string

func InitPaths() {
	AppDir = appdir.New("xbvr").UserConfig()

	ImgDir = filepath.Join(AppDir, "imageproxy")
	CacheDir = filepath.Join(AppDir, "cache")
	BinDir = filepath.Join(AppDir, "bin")
	IndexDir = filepath.Join(AppDir, "search")

	_ = os.MkdirAll(AppDir, os.ModePerm)
	_ = os.MkdirAll(ImgDir, os.ModePerm)
	_ = os.MkdirAll(CacheDir, os.ModePerm)
	_ = os.MkdirAll(BinDir, os.ModePerm)
	_ = os.MkdirAll(IndexDir, os.ModePerm)
}
