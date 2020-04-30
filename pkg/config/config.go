package config

import (
	"encoding/json"

	"github.com/creasty/defaults"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

type Object struct {
	Server struct {
		BindAddress string `default:"0.0.0.0" json:"bindAddress"`
		Port        int    `default:"9999" json:"port"`
	} `json:"server"`
	Security struct {
		Username string `default:"" json:"username"`
		Password string `default:"" json:"password"`
	} `json:"security"`
	Interface struct {
		DLNA struct {
			Enabled bool `default:"true" json:"enabled"`
		} `json:"dlna"`
		DeoVR struct {
			Enabled bool `default:"true" json:"enabled"`
		} `json:"deovr"`
	} `json:"interface"`
	Library struct {
		Preview struct {
			Enabled       bool    `default:"true" json:"enabled"`
			StartTime     int     `default:"10" json:"startTime"`
			SnippetLength float64 `default:"0.4" json:"snippetLength"`
			SnippetAmount int     `default:"20" json:"snippetAmount"`
		} `json:"preview"`
	} `json:"library"`
	Cron struct {
		ScrapeContent string `default:"" json:"scrapeContent"`
		RescanLibrary string `default:"" json:"rescanLibrary"`
	} `json:"cron"`
}

var Config Object

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
}
