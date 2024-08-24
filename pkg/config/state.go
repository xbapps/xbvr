package config

import (
	"encoding/json"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type ObjectState struct {
	Server struct {
		BoundIP []string `json:"bound_ip"`
	} `json:"server"`
	Web struct {
		TagSort             string `json:"tagSort"`
		SceneHidden         bool   `json:"sceneHidden"`
		SceneWatchlist      bool   `json:"sceneWatchlist"`
		SceneFavourite      bool   `json:"sceneFavourite"`
		SceneWishlist       bool   `json:"sceneWishlist"`
		SceneWatched        bool   `json:"sceneWatched"`
		SceneEdit           bool   `json:"sceneEdit"`
		SceneDuration       bool   `json:"sceneDuration"`
		SceneCuepoint       bool   `json:"sceneCuepoint"`
		ShowHspFile         bool   `json:"showHspFile"`
		ShowSubtitlesFile   bool   `json:"showSubtitlesFile"`
		SceneTrailerlist    bool   `json:"sceneTrailerlist"`
		ShowScriptHeatmap   bool   `json:"showScriptHeatmap"`
		ShowAllHeatmaps     bool   `json:"showAllHeatmaps"`
		ShowOpenInNewWindow bool   `json:"showOpenInNewWindow"`
		UpdateCheck         bool   `json:"updateCheck"`
		IsAvailOpacity      int    `json:"isAvailOpacity"`
	} `json:"web"`
	DLNA struct {
		Running  bool     `json:"running"`
		Images   []string `json:"images"`
		RecentIP []string `json:"recentIp"`
	} `json:"dlna"`
	CacheSize struct {
		Images      int64 `json:"images"`
		Previews    int64 `json:"previews"`
		SearchIndex int64 `json:"searchIndex"`
	} `json:"cacheSize"`
}

var State ObjectState

func LoadState() {
	db, _ := models.GetDB()
	defer db.Close()

	var obj models.KV
	err := db.Where(&models.KV{Key: "state"}).First(&obj).Error
	if err == nil {
		if err := json.Unmarshal([]byte(obj.Value), &State); err != nil {
			common.Log.Error("Failed to load state from database")
		}
	}
}

func SaveState() {
	data, err := json.Marshal(State)
	if err == nil {
		obj := models.KV{Key: "state", Value: string(data)}
		obj.Save()
		common.Log.Info("Saved state")
	}
}
