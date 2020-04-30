package xbvr

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/xbapps/xbvr/pkg/config"
)

var cronInstance *cron.Cron

func SetupCron() {
	cronInstance := cron.New()
	cronInstance.AddFunc("@every 20s", checkForDeadSession)
	cronInstance.AddFunc(fmt.Sprintf("@every %vh", config.Config.Cron.ScrapeContentInterval), scrapeCron)
	cronInstance.AddFunc(fmt.Sprintf("@every %vh", config.Config.Cron.RescanLibraryInterval), RescanVolumes)
	cronInstance.Start()
}

func scrapeCron() {
	Scrape("_enabled")
}
