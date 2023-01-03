package config

import (
	"encoding/json"

	"github.com/creasty/defaults"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type CronSchedule struct {
	Enabled         bool `default:"true" json:"enabled"`
	HourInterval    int  `json:"hourInterval"`
	UseRange        bool `default:"false" json:"useRange"`
	MinuteStart     int  `default:"0" json:"minuteStart"`
	HourStart       int  `default:"0" json:"hourStart"`
	HourEnd         int  `default:"23" json:"hourEnd"`
	RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
}

type ObjectConfig struct {
	Server struct {
		BindAddress string `default:"0.0.0.0" json:"bindAddress"`
		Port        int    `default:"9999" json:"port"`
	} `json:"server"`
	Security struct {
		Username string `default:"" json:"username"`
		Password string `default:"" json:"password"`
	} `json:"security"`
	Web struct {
		TagSort          string `default:"by-tag-count" json:"tagSort"`
		SceneWatchlist   bool   `default:"true" json:"sceneWatchlist"`
		SceneFavourite   bool   `default:"true" json:"sceneFavourite"`
		SceneWatched     bool   `default:"false" json:"sceneWatched"`
		SceneEdit        bool   `default:"false" json:"sceneEdit"`
		SceneCuepoint    bool   `default:"true" json:"sceneCuepoint"`
		ShowHspFile      bool   `default:"true" json:"showHspFile"`
		SceneTrailerlist bool   `default:"true" json:"sceneTrailerlist"`
		UpdateCheck      bool   `default:"true" json:"updateCheck"`
	} `json:"web"`
	Vendor struct {
		TPDB struct {
			ApiToken string `default:"" json:"apiToken"`
		} `json:"tpdb"`
	} `json:"vendor"`
	Interfaces struct {
		DLNA struct {
			Enabled      bool     `default:"true" json:"enabled"`
			ServiceName  string   `default:"XBVR" json:"serviceName"`
			ServiceImage string   `default:"default" json:"serviceImage"`
			AllowedIP    []string `default:"[]" json:"allowedIp"`
		} `json:"dlna"`
		DeoVR struct {
			Enabled        bool   `default:"true" json:"enabled"`
			AuthEnabled    bool   `default:"false" json:"auth_enabled"`
			RenderHeatmaps bool   `default:"false" json:"render_heatmaps"`
			TrackWatchTime bool   `default:"true" json:"track_watch_time"`
			RemoteEnabled  bool   `default:"false" json:"remote_enabled"`
			Username       string `default:"" json:"username"`
			Password       string `default:"" json:"password"`
		} `json:"deovr"`
		Heresphere struct {
			AllowFileDeletes        bool `default:"false" json:"allow_file_deletes"`
			AllowRatingUpdates      bool `default:"false" json:"allow_rating_updates"`
			AllowFavoriteUpdates    bool `default:"false" json:"allow_favorite_updates"`
			AllowHspData            bool `default:"false" json:"allow_hsp_data"`
			AllowTagUpdates         bool `default:"false" json:"allow_tag_updates"`
			AllowCuepointUpdates    bool `default:"false" json:"allow_cuepoint_updates"`
			AllowWatchlistUpdates   bool `default:"false" json:"allow_watchlist_updates"`
			MultitrackCuepoints     bool `default:"true" json:"multitrack_cuepoints"`
			MultitrackCastCuepoints bool `default:"true" json:"multitrack_cast_cuepoints"`
			RetainNonHSPCuepoints   bool `default:"true" json:"retain_non_hsp_cuepoints"`
		} `json:"heresphere"`
	} `json:"interfaces"`
	Library struct {
		Preview struct {
			Enabled       bool    `default:"true" json:"enabled"`
			StartTime     int     `default:"10" json:"startTime"`
			SnippetLength float64 `default:"0.4" json:"snippetLength"`
			SnippetAmount int     `default:"20" json:"snippetAmount"`
			Resolution    int     `default:"400" json:"resolution"`
			ExtraSnippet  bool    `default:"false" json:"extraSnippet"`
		} `json:"preview"`
	} `json:"library"`
	Cron struct {
		RescrapeSchedule struct {
			Enabled         bool `default:"true" json:"enabled"`
			HourInterval    int  `default:"12" json:"hourInterval"`
			UseRange        bool `default:"false" json:"useRange"`
			MinuteStart     int  `default:"0" json:"minuteStart"`
			HourStart       int  `default:"0" json:"hourStart"`
			HourEnd         int  `default:"23" json:"hourEnd"`
			RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
		} `json:"rescrapeSchedule"`
		RescanSchedule struct {
			Enabled         bool `default:"true" json:"enabled"`
			HourInterval    int  `default:"2" json:"hourInterval"`
			UseRange        bool `default:"false" json:"useRange"`
			MinuteStart     int  `default:"0" json:"minuteStart"`
			HourStart       int  `default:"0" json:"hourStart"`
			HourEnd         int  `default:"23" json:"hourEnd"`
			RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
		} `json:"rescanSchedule"`
		PreviewSchedule struct {
			Enabled         bool `default:"false" json:"enabled"`
			HourInterval    int  `default:"2" json:"hourInterval"`
			UseRange        bool `default:"false" json:"useRange"`
			MinuteStart     int  `default:"0" json:"minuteStart"`
			HourStart       int  `default:"0" json:"hourStart"`
			HourEnd         int  `default:"23" json:"hourEnd"`
			RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
		} `json:"previewSchedule"`
	} `json:"cron"`
}

var Config ObjectConfig
var RecentIPAddresses []string

func LoadConfig() {
	db, _ := models.GetDB()
	defer db.Close()

	var obj models.KV
	err := db.Where(&models.KV{Key: "config"}).First(&obj).Error
	if err == nil {
		if err := json.Unmarshal([]byte(obj.Value), &Config); err != nil {
			common.Log.Error("Failed to load config from database")
		}
	}
}

func SaveConfig() {
	data, err := json.Marshal(Config)
	if err == nil {
		obj := models.KV{Key: "config", Value: string(data)}
		obj.Save()
		common.Log.Info("Saved config")
	}
}

func init() {
	defaults.Set(&Config)
	RecentIPAddresses = []string{}
}
