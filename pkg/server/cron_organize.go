package server

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/api"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/organize"
	"github.com/xbapps/xbvr/pkg/session"
)

// organizeCheckEvery is how often an interval-based organize schedule checks whether a
// run is due. A run skipped because a player session is open, or because another
// organize job is running, is tried again at the next check.
const organizeCheckEvery = 10 * time.Minute

// organizeLastRunKey is the KV key holding the time of the last scheduled organize run.
const organizeLastRunKey = "organize-schedule-last-run"

var (
	startOrganize    = organize.Start
	hasActiveSession = session.HasActiveSession
)

// intervalSchedule decides when an interval-based task is due. The interval counts from
// the last run recorded through load/save, which persist across restarts, rather than
// from process start.
type intervalSchedule struct {
	every time.Duration
	load  func() (time.Time, error)
	save  func(time.Time)
}

// anchor records now as the last run when none is recorded yet, so a newly enabled
// schedule first runs one interval later.
func (s intervalSchedule) anchor(now time.Time) {
	if last, err := s.load(); err == nil && last.IsZero() {
		s.save(now)
	}
}

// next returns when the task is next due; ok is false when the last run can't be read.
func (s intervalSchedule) next() (t time.Time, ok bool) {
	last, err := s.load()
	if err != nil {
		return time.Time{}, false
	}
	return last.Add(s.every), true
}

// runIfDue calls start when the task is due and records the run when start reports that
// it began. Returns whether a run began.
func (s intervalSchedule) runIfDue(now time.Time, start func() bool) bool {
	next, ok := s.next()
	if !ok || now.Before(next) || !start() {
		return false
	}
	s.save(now)
	return true
}

func organizeInterval(hours int) time.Duration {
	if hours < 1 {
		hours = 1
	}
	return time.Duration(hours) * time.Hour
}

func newOrganizeSchedule(hours int) intervalSchedule {
	return intervalSchedule{every: organizeInterval(hours), load: loadOrganizeLastRun, save: saveOrganizeLastRun}
}

func setupOrganizeCron(schedule config.CronSchedule) {
	if schedule.UseRange {
		spec := formatCronSchedule(schedule)
		log.Println(fmt.Sprintf("Setup Organize Task %v", spec))
		organizeTask, _ = cronInstance.AddFunc(spec, func() {
			organizeCron()
			log.Println(fmt.Sprintf("Next Organize Task at %v", cronInstance.Entry(organizeTask).Next))
		})
		return
	}
	s := newOrganizeSchedule(schedule.HourInterval)
	s.anchor(time.Now())
	if next, ok := s.next(); ok {
		log.Println(fmt.Sprintf("Setup Organize Task every %v, next due %v", s.every, next.Local().Format(time.RFC3339)))
	} else {
		log.Println(fmt.Sprintf("Setup Organize Task every %v", s.every))
	}
	organizeTask, _ = cronInstance.AddFunc(fmt.Sprintf("@every %v", organizeCheckEvery), func() {
		if s.runIfDue(time.Now(), startScheduledOrganize) {
			if next, ok := s.next(); ok {
				log.Println(fmt.Sprintf("Next Organize Task due %v", next.Local().Format(time.RFC3339)))
			}
		}
	})
}

// organizeCron starts a scheduled organize run and records it as the last run.
func organizeCron() {
	if startScheduledOrganize() {
		saveOrganizeLastRun(time.Now())
	}
}

// startScheduledOrganize starts an organize run with the saved organize options unless a
// player session is open or another organize job is running. The run applies its changes
// when "apply on schedule" is on; otherwise it only works out the plan for review.
// Returns whether a run started.
func startScheduledOrganize() bool {
	if hasActiveSession() {
		return false
	}
	apply := config.Config.Organize.ScheduleApply
	if !startOrganize(api.OrganizeOptionsFromConfig(!apply, 0)) {
		log.Println("Organize Task: another organize job is running; will try again")
		return false
	}
	if apply {
		log.Println("Organize Task started, applying changes")
	} else {
		log.Println("Organize Task started, preview only")
	}
	return true
}

func loadOrganizeLastRun() (time.Time, error) {
	db, err := models.GetDB()
	if err != nil {
		return time.Time{}, err
	}
	defer db.Close()
	var kv models.KV
	if err := db.Where(&models.KV{Key: organizeLastRunKey}).First(&kv).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339, kv.Value)
	if err != nil {
		return time.Time{}, nil
	}
	return t, nil
}

func saveOrganizeLastRun(t time.Time) {
	kv := models.KV{Key: organizeLastRunKey, Value: t.UTC().Format(time.RFC3339)}
	kv.Save()
}
