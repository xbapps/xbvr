package config

import (
	"encoding/json"

	"github.com/creasty/defaults"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

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
		TagSort   string `default:"by-tag-count" json:"tagSort"`
		SceneEdit bool   `default:"false" json:"sceneEdit"`
	} `json:"web"`
	Interfaces struct {
		DLNA struct {
			Enabled      bool     `default:"true" json:"enabled"`
			ServiceName  string   `default:"XBVR" json:"serviceName"`
			ServiceImage string   `default:"default" json:"serviceImage"`
			AllowedIP    []string `default:"[]" json:"allowedIp"`
		} `json:"dlna"`
		DeoVR struct {
			Enabled       bool   `default:"true" json:"enabled"`
			AuthEnabled   bool   `default:"false" json:"auth_enabled"`
			RemoteEnabled bool   `default:"false" json:"remote_enabled"`
			Username      string `default:"" json:"username"`
			Password      string `default:"" json:"password"`
		} `json:"deovr"`
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
		ScrapeContentInterval int `default:"12" json:"scrapeContentInt"`
		RescanLibraryInterval int `default:"2" json:"rescanLibraryInt"`
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
