package server

import (
	"errors"
	"testing"
	"time"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/organize"
)

type fakeLastRun struct {
	t     time.Time
	err   error
	saves int
}

func (f *fakeLastRun) schedule(every time.Duration) intervalSchedule {
	return intervalSchedule{
		every: every,
		load:  func() (time.Time, error) { return f.t, f.err },
		save:  func(t time.Time) { f.t = t; f.saves++ },
	}
}

var t0 = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestIntervalAnchorsFirstRunOneIntervalAfterEnabling(t *testing.T) {
	f := &fakeLastRun{}
	s := f.schedule(24 * time.Hour)
	s.anchor(t0)
	if !f.t.Equal(t0) {
		t.Fatalf("anchor = %v, want %v", f.t, t0)
	}
	started := 0
	start := func() bool { started++; return true }
	if s.runIfDue(t0.Add(23*time.Hour), start) || started != 0 {
		t.Fatal("ran before the interval was up")
	}
	if !s.runIfDue(t0.Add(24*time.Hour), start) || started != 1 {
		t.Fatal("didn't run once the interval was up")
	}
}

func TestIntervalAnchorKeepsRecordedRun(t *testing.T) {
	last := t0.Add(-20 * time.Hour)
	f := &fakeLastRun{t: last}
	f.schedule(24 * time.Hour).anchor(t0)
	if !f.t.Equal(last) || f.saves != 0 {
		t.Fatalf("anchor replaced the recorded run: %v", f.t)
	}
}

func TestIntervalSurvivesRestart(t *testing.T) {
	// Last run 20h ago; a restart builds a new schedule from the stored time, so the
	// run is still due 4h from now rather than 24h after the restart.
	f := &fakeLastRun{t: t0.Add(-20 * time.Hour)}
	s := f.schedule(24 * time.Hour)
	s.anchor(t0)
	next, ok := s.next()
	if !ok || !next.Equal(t0.Add(4*time.Hour)) {
		t.Fatalf("next = %v (ok %v), want %v", next, ok, t0.Add(4*time.Hour))
	}
}

func TestIntervalOverdueRunsAtFirstCheck(t *testing.T) {
	f := &fakeLastRun{t: t0.Add(-72 * time.Hour)}
	if !f.schedule(24*time.Hour).runIfDue(t0, func() bool { return true }) {
		t.Fatal("overdue run didn't start")
	}
	if !f.t.Equal(t0) {
		t.Fatalf("last run = %v, want %v", f.t, t0)
	}
}

func TestIntervalSkippedRunIsRetried(t *testing.T) {
	f := &fakeLastRun{t: t0.Add(-24 * time.Hour)}
	s := f.schedule(24 * time.Hour)
	if s.runIfDue(t0, func() bool { return false }) {
		t.Fatal("reported a run that didn't start")
	}
	if f.saves != 0 {
		t.Fatal("recorded a run that didn't start")
	}
	if !s.runIfDue(t0.Add(organizeCheckEvery), func() bool { return true }) {
		t.Fatal("skipped run wasn't retried at the next check")
	}
}

func TestIntervalUnreadableLastRunDoesNotRun(t *testing.T) {
	f := &fakeLastRun{err: errors.New("database is locked")}
	s := f.schedule(24 * time.Hour)
	s.anchor(t0)
	if f.saves != 0 {
		t.Fatal("anchored over an unreadable last run")
	}
	if s.runIfDue(t0, func() bool { t.Fatal("started"); return true }) {
		t.Fatal("ran with an unreadable last run")
	}
}

func TestOrganizeIntervalFloor(t *testing.T) {
	if got := organizeInterval(0); got != time.Hour {
		t.Fatalf("organizeInterval(0) = %v, want 1h", got)
	}
}

func stubOrganize(t *testing.T, active, startOK bool) *[]organize.Options {
	t.Helper()
	var got []organize.Options
	prevStart, prevActive, prevApply := startOrganize, hasActiveSession, config.Config.Organize.ScheduleApply
	startOrganize = func(o organize.Options) bool { got = append(got, o); return startOK }
	hasActiveSession = func() bool { return active }
	t.Cleanup(func() {
		startOrganize, hasActiveSession = prevStart, prevActive
		config.Config.Organize.ScheduleApply = prevApply
	})
	return &got
}

func TestScheduledOrganizeAppliesWhenEnabled(t *testing.T) {
	got := stubOrganize(t, false, true)
	config.Config.Organize.ScheduleApply = true
	if !startScheduledOrganize() {
		t.Fatal("didn't start")
	}
	if len(*got) != 1 || (*got)[0].DryRun {
		t.Fatalf("options = %+v, want one applying run", *got)
	}
}

func TestScheduledOrganizePreviewsByDefault(t *testing.T) {
	got := stubOrganize(t, false, true)
	config.Config.Organize.ScheduleApply = false
	startScheduledOrganize()
	if len(*got) != 1 || !(*got)[0].DryRun {
		t.Fatalf("options = %+v, want one preview run", *got)
	}
}

func TestScheduledOrganizeSkipsDuringSession(t *testing.T) {
	got := stubOrganize(t, true, true)
	config.Config.Organize.ScheduleApply = true
	if startScheduledOrganize() || len(*got) != 0 {
		t.Fatal("started while a player session was open")
	}
}

func TestScheduledOrganizeReportsBusy(t *testing.T) {
	stubOrganize(t, false, false)
	if startScheduledOrganize() {
		t.Fatal("reported a start while another organize job was running")
	}
}
