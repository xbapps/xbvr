package server

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
)

var cronInstance *cron.Cron

func SetupCron() {
	cronInstance := cron.New()
	cronInstance.AddFunc("@every 2s", session.CheckForDeadSession)
	cronInstance.AddFunc("@every 6h", tasks.CalculateCacheSizes)
	cronInstance.AddFunc(fmt.Sprintf("@every %vh", config.Config.Cron.ScrapeContentInterval), scrapeCron)
	cronInstance.AddFunc(fmt.Sprintf("@every %vh", config.Config.Cron.RescanLibraryInterval), tasks.RescanVolumes)
	cronInstance.Start()

	go tasks.CalculateCacheSizes()
}

func scrapeCron() {
	tasks.Scrape("_enabled")
}
