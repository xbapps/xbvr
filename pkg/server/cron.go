package server

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
)

var cronInstance *cron.Cron
var rescrapTask cron.EntryID
var rescanTask cron.EntryID

func SetupCron() {
	cronInstance = cron.New()
	cronInstance.AddFunc("@every 2s", session.CheckForDeadSession)
	cronInstance.AddFunc("@every 6h", tasks.CalculateCacheSizes)
	if config.Config.Cron.RescrapeSchedule.Enabled {
		log.Println(fmt.Sprintf("Setup Rescrape Task %v", formatCronSchedule(config.CronSchedule(config.Config.Cron.RescrapeSchedule))))
		rescrapTask, _ = cronInstance.AddFunc(formatCronSchedule(config.CronSchedule(config.Config.Cron.RescrapeSchedule)), scrapeCron)
	}
	if config.Config.Cron.RescanSchedule.Enabled {
		log.Println(fmt.Sprintf("Setup Rescan Task %v", formatCronSchedule(config.CronSchedule(config.Config.Cron.RescanSchedule))))
		rescanTask, _ = cronInstance.AddFunc(formatCronSchedule(config.CronSchedule(config.Config.Cron.RescanSchedule)), rescanCron)
	}
	cronInstance.Start()

	go tasks.CalculateCacheSizes()

	log.Println(fmt.Sprintf("Next Rescrape Task at %v", cronInstance.Entry(rescrapTask).Next))
	log.Println(fmt.Sprintf("Next Rescan Task at %v", cronInstance.Entry(rescanTask).Next))
}

func scrapeCron() {
	if !session.HasActiveSession() {
		tasks.Scrape("_enabled")
	}
	log.Println(fmt.Sprintf("Next Rescrape Task at %v", cronInstance.Entry(rescrapTask).Next))
}

func rescanCron() {
	if !session.HasActiveSession() {
		tasks.RescanVolumes()
	}
	log.Println(fmt.Sprintf("Next Rescan Task at %v", cronInstance.Entry(rescanTask).Next))
}
func formatCronSchedule(schedule config.CronSchedule) string {
	// 	this routine will format a crontab range description, https://crontab.guru is a good tool to decode the range description generated
	// 	if the start hour > end hour then the time range will extend across midnight into the next day
	//		to achieve this with cron you create a range from the start until midnight and then a second from from midnight to the end time
	//		we need to calculate the start time for the range after midnight to make sure we still get the right iterval
	hourInterval := ""
	formattedHourSchedule := ""

	if !schedule.UseRange {
		return fmt.Sprintf("@every %vh", schedule.HourInterval)
	}

	if schedule.HourInterval > 0 {
		hourInterval = fmt.Sprintf("/%v", schedule.HourInterval)
	}
	if schedule.HourStart > schedule.HourEnd { // if the start > end, time range goes over midnight into the next day
		afterMidnightStart := (schedule.HourInterval - ((24 - schedule.HourStart) % schedule.HourInterval)) % schedule.HourInterval // calculate what time after midnight to restart
		if afterMidnightStart <= schedule.HourEnd {
			// schedule the range needed to start after midnight
			formattedHourSchedule = fmt.Sprintf("%v-%v%v,%v-23%v", afterMidnightStart, schedule.HourEnd, hourInterval, schedule.HourStart, hourInterval)
		} else {
			// the interval was too big to schedule after midnight before reaching the end time, so only create the pre midnight range
			formattedHourSchedule = fmt.Sprintf("%v-23%v", schedule.HourStart, hourInterval)
		}
	} else {
		formattedHourSchedule = fmt.Sprintf("%v-%v%v", schedule.HourStart, schedule.HourEnd, hourInterval)
	}
	return fmt.Sprintf("%v %v * * *", schedule.MinuteStart, formattedHourSchedule)
}
