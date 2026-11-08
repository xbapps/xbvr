package organize

import (
	"sync"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var (
	runMu   sync.Mutex
	running bool
	last    *Result

	// busy is shared by the organize runner and the duplicate analyzer so the two never
	// run at once — they touch the same files on disk. Held for a job's whole lifetime.
	busy sync.Mutex
)

// Start launches a run in the background (preview if opts.DryRun). Returns false if a
// run (or a duplicate analysis) is already in progress.
func Start(opts Options) bool {
	if !busy.TryLock() {
		return false
	}
	runMu.Lock()
	running = true
	runMu.Unlock()

	go func() {
		defer busy.Unlock()
		var r *Result
		db, err := models.GetDB()
		if err != nil {
			common.Log.Errorf("organize: db open failed: %v", err)
		} else {
			r = Run(db, opts)
			db.Close()
		}
		runMu.Lock()
		last = r
		running = false
		runMu.Unlock()
	}()
	return true
}

// Status reports whether a run is in progress and the most recent result.
func Status() (bool, *Result) {
	runMu.Lock()
	defer runMu.Unlock()
	return running, last
}
