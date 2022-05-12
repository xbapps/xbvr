package tasks

import (
	"io/fs"
	"strings"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/ui"
)

func UpdateState() {
	// DLNA
	var dlnaImages []string
	fs.WalkDir(ui.Assets, "dist/dlna", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			dlnaImages = append(dlnaImages, path)
		}
		return nil
	})

	config.State.DLNA.Images = make([]string, 0)
	for _, v := range dlnaImages {
		config.State.DLNA.Images = append(config.State.DLNA.Images, strings.Replace(strings.Split(v, "/")[2], ".png", "", -1))
	}

	config.State.DLNA.Running = IsDMSStarted()
	config.State.DLNA.RecentIP = config.RecentIPAddresses

	config.SaveState()
}

func CalculateCacheSizes() {
	config.State.CacheSize.Images, _ = common.DirSize(common.ImgDir)
	config.State.CacheSize.Previews, _ = common.DirSize(common.VideoPreviewDir)
	config.State.CacheSize.SearchIndex, _ = common.DirSize(common.IndexDirV2)

	config.SaveState()
}
