package llmdraft

import (
	"context"
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// BatchStatus reports progress of a batch run over unmatched files.
type BatchStatus struct {
	Running       bool      `json:"running"`
	Total         int       `json:"total"`
	Done          int       `json:"done"`
	DraftsCreated int       `json:"drafts_created"`
	Errors        int       `json:"errors"`
	Current       string    `json:"current"`
	LastError     string    `json:"last_error"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
}

var (
	batchMu     sync.Mutex
	batchState  BatchStatus
	batchCancel context.CancelFunc
)

// Status returns the current batch progress.
func Status() BatchStatus {
	batchMu.Lock()
	defer batchMu.Unlock()
	return batchState
}

// StopBatch cancels a running batch after the file in progress.
func StopBatch() {
	batchMu.Lock()
	defer batchMu.Unlock()
	if batchCancel != nil {
		batchCancel()
	}
}

// StartBatch searches for drafts for every unmatched video file in the background. Files that
// already have drafts or were already searched are skipped unless force is set, in which case
// their unreviewed search drafts are replaced; limit 0 means no limit. It returns false
// when a batch is already running.
func StartBatch(userAgent string, force bool, limit int) (bool, error) {
	svc, err := NewService(userAgent)
	if err != nil {
		return false, err
	}

	batchMu.Lock()
	if batchState.Running {
		batchMu.Unlock()
		return false, nil
	}
	ids := unmatchedFileIDs(force, limit)
	ctx, cancel := context.WithCancel(context.Background())
	batchCancel = cancel
	batchState = BatchStatus{Running: true, Total: len(ids), StartedAt: time.Now()}
	batchMu.Unlock()

	go func() {
		defer cancel()
		for _, id := range ids {
			if ctx.Err() != nil {
				break
			}
			setCurrent(id)
			res, err := svc.SuggestForFile(ctx, id, force)

			batchMu.Lock()
			batchState.Done++
			if err != nil {
				batchState.Errors++
				batchState.LastError = err.Error()
			} else {
				for _, r := range res.Results {
					if r.Draft != nil {
						batchState.DraftsCreated++
					}
				}
			}
			batchMu.Unlock()
		}
		batchMu.Lock()
		batchState.Running = false
		batchState.Current = ""
		batchState.FinishedAt = time.Now()
		common.Log.Infof("LLM scraper: batch finished, %d/%d files, %d drafts, %d errors",
			batchState.Done, batchState.Total, batchState.DraftsCreated, batchState.Errors)
		batchMu.Unlock()
	}()
	return true, nil
}

func setCurrent(fileID uint) {
	name := ""
	if f, err := LoadFileInfo(fileID); err == nil {
		name = f.Filename
	}
	batchMu.Lock()
	batchState.Current = name
	batchMu.Unlock()
}

// unmatchedFileIDs lists unmatched video files, newest first.
func unmatchedFileIDs(force bool, limit int) []uint {
	db, _ := models.GetDB()
	defer db.Close()
	q := db.Model(&models.File{}).Where("scene_id = 0 AND type = ?", "video")
	if !force {
		q = q.Where("id NOT IN (SELECT DISTINCT file_id FROM draft_scenes)").
			Where("id NOT IN (SELECT file_id FROM file_match_contexts WHERE last_searched_at IS NOT NULL)")
	}
	q = q.Order("created_time desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var ids []uint
	q.Pluck("id", &ids)
	return ids
}
