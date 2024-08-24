package config

import (
	"encoding/json"
	"time"

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
		TagSort             string `default:"by-tag-count" json:"tagSort"`
		SceneHidden         bool   `default:"true" json:"sceneHidden"`
		SceneWatchlist      bool   `default:"true" json:"sceneWatchlist"`
		SceneFavourite      bool   `default:"true" json:"sceneFavourite"`
		SceneWishlist       bool   `default:"true" json:"sceneWishlist"`
		SceneWatched        bool   `default:"false" json:"sceneWatched"`
		SceneEdit           bool   `default:"false" json:"sceneEdit"`
		SceneDuration       bool   `default:"false" json:"sceneDuration"`
		SceneCuepoint       bool   `default:"true" json:"sceneCuepoint"`
		ShowHspFile         bool   `default:"true" json:"showHspFile"`
		ShowSubtitlesFile   bool   `default:"true" json:"showSubtitlesFile"`
		SceneTrailerlist    bool   `default:"true" json:"sceneTrailerlist"`
		ShowScriptHeatmap   bool   `default:"true" json:"showScriptHeatmap"`
		ShowAllHeatmaps     bool   `default:"false" json:"showAllHeatmaps"`
		ShowOpenInNewWindow bool   `default:"true" json:"showOpenInNewWindow"`
		UpdateCheck         bool   `default:"true" json:"updateCheck"`
		IsAvailOpacity      int    `default:"40" json:"isAvailOpacity"`
	} `json:"web"`
	Advanced struct {
		ShowInternalSceneId          bool      `default:"false" json:"showInternalSceneId"`
		ShowHSPApiLink               bool      `default:"false" json:"showHSPApiLink"`
		ShowSceneSearchField         bool      `default:"false" json:"showSceneSearchField"`
		StashApiKey                  string    `default:"" json:"stashApiKey"`
		ScrapeActorAfterScene        bool      `default:"true" json:"scrapeActorAfterScene"`
		UseImperialEntry             bool      `default:"false" json:"useImperialEntry"`
		ProgressTimeInterval         int       `default:"15" json:"progressTimeInterval"`
		LinkScenesAfterSceneScraping bool      `default:"true" json:"linkScenesAfterSceneScraping"`
		UseAltSrcInFileMatching      bool      `default:"true" json:"useAltSrcInFileMatching"`
		UseAltSrcInScriptFilters     bool      `default:"true" json:"useAltSrcInScriptFilters"`
		IgnoreReleasedBefore         time.Time `json:"ignoreReleasedBefore"`
	} `json:"advanced"`
	Funscripts struct {
		ScrapeFunscripts bool `default:"false" json:"scrapeFunscripts"`
	} `json:"funscripts"`
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
		Players struct {
			VideoSortSeq    string `default:"" json:"video_sort_seq"`
			ScriptSortSeq   string `default:"" json:"script_sort_seq"`
			SubtitleSortSeq string `default:"" json:"subtitle_sort_seq"`
		} `json:"players"`
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
		ActorRescrapeSchedule struct {
			Enabled         bool `default:"false" json:"enabled"`
			HourInterval    int  `default:"12" json:"hourInterval"`
			UseRange        bool `default:"false" json:"useRange"`
			MinuteStart     int  `default:"0" json:"minuteStart"`
			HourStart       int  `default:"0" json:"hourStart"`
			HourEnd         int  `default:"23" json:"hourEnd"`
			RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
		} `json:"actorRescrapeSchedule"`
		StashdbRescrapeSchedule struct {
			Enabled         bool `default:"false" json:"enabled"`
			HourInterval    int  `default:"12" json:"hourInterval"`
			UseRange        bool `default:"false" json:"useRange"`
			MinuteStart     int  `default:"0" json:"minuteStart"`
			HourStart       int  `default:"0" json:"hourStart"`
			HourEnd         int  `default:"23" json:"hourEnd"`
			RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
		} `json:"stashdbRescrapeSchedule"`
		LinkScenesSchedule struct {
			Enabled         bool `default:"false" json:"enabled"`
			HourInterval    int  `default:"12" json:"hourInterval"`
			UseRange        bool `default:"false" json:"useRange"`
			MinuteStart     int  `default:"0" json:"minuteStart"`
			HourStart       int  `default:"0" json:"hourStart"`
			HourEnd         int  `default:"23" json:"hourEnd"`
			RunAtStartDelay int  `default:"0" json:"runAtStartDelay"`
		} `json:"linkScenesSchedule"`
	} `json:"cron"`
	Storage struct {
		MatchOhash bool `default:"false" json:"match_ohash"`
	} `json:"storage"`
	ScraperSettings struct {
		TMWVRNet struct {
			TmwMembersDomain string `default:"members.tmwvrnet.com" json:"tmwMembersDomain"`
		} `json:"tmwvrnet"`
		Javr struct {
			JavrScraper string `default:"javdatabase" json:"javrScraper"`
		} `json:"javr"`
	} `json:"scraper_settings"`
}

var (
	Config            ObjectConfig
	RecentIPAddresses []string
)

func LoadConfig() {
	db, _ := models.GetDB()
	defer db.Close()

	var obj models.KV
	err := db.Where(&models.KV{Key: "config"}).First(&obj).Error
	if err == nil {
		if err := json.Unmarshal([]byte(obj.Value), &Config); err != nil {
			common.Log.Error("Failed to load config from database")
		}
		if common.WebPort != 0 && common.WebPort != Config.Server.Port {
			Config.Server.Port = common.WebPort
			SaveConfig()
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
