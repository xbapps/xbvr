package tasks

import (
	"strings"

	"github.com/xbapps/xbvr/pkg/assets"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
)

func UpdateState() {
	// DLNA
	config.State.DLNA.Running = IsDMSStarted()
	config.State.DLNA.RecentIP = config.RecentIPAddresses
	dlnaImages, _ := assets.WalkDirs("dlna", false)

	config.State.DLNA.Images = make([]string, 0)
	for _, v := range dlnaImages {
		config.State.DLNA.Images = append(config.State.DLNA.Images, strings.Replace(strings.Split(v, "/")[1], ".png", "", -1))
	}

	config.SaveState()
}

func CalculateCacheSizes() {
	config.State.CacheSize.Images, _ = common.DirSize(common.ImgDir)
	config.State.CacheSize.Previews, _ = common.DirSize(common.VideoPreviewDir)
	config.State.CacheSize.SearchIndex, _ = common.DirSize(common.IndexDirV2)

	config.SaveState()
}
