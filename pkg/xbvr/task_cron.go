package xbvr

import (
	"github.com/robfig/cron/v3"
)

var cronInstance *cron.Cron

func SetupCron() {
	cronInstance := cron.New()
	cronInstance.AddFunc("@every 20s", checkForDeadSession)
	cronInstance.Start()
}
