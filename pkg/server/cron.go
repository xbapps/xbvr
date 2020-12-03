package server

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/xbapps/xbvr/pkg/api"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/tasks"
)

var cronInstance *cron.Cron

func SetupCron() {
	cronInstance := cron.New()
	cronInstance.AddFunc("@every 20s", api.CheckForDeadSession)
	cronInstance.AddFunc(fmt.Sprintf("@every %vh", config.Config.Cron.ScrapeContentInterval), scrapeCron)
	cronInstance.AddFunc(fmt.Sprintf("@every %vh", config.Config.Cron.RescanLibraryInterval), tasks.RescanVolumes)
	cronInstance.Start()
}

func scrapeCron() {
	tasks.Scrape("_enabled")
}
