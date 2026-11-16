package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
)

// Draft scene statuses.
const (
	DraftStatusDraft     = "draft"     // proposed, awaiting review
	DraftStatusSaved     = "saved"     // promoted to a real scene
	DraftStatusRejected  = "rejected"  // discarded by the user; kept so it is not proposed again
	DraftStatusDismissed = "dismissed" // another draft for the same file was saved
)

// DraftScene is a scene proposed by the LLM scraper from a web page, held for review before it is
// added to the library. SceneJSON holds a ScrapedScene, the same structure every site scraper
// produces, so saving a draft goes through the normal scene import path.
type DraftScene struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	FileID    uint   `gorm:"index" json:"file_id"` // file the draft was proposed for; 0 when scraped directly
	SourceURL string `gorm:"size:1000" json:"source_url"`
	Query     string `gorm:"size:1000" json:"query"` // web search query that found the page, if any
	Status    string `gorm:"size:20;index" json:"status"`

	MatchConfidence float64 `json:"match_confidence"` // LLM's confidence that the page is the file's scene
	PageKind        string  `gorm:"size:40" json:"page_kind"`
	Reason          string  `sql:"type:text" json:"reason"`
	Model           string  `gorm:"size:200" json:"model"`

	SceneJSON string `sql:"type:text" json:"-"`
	SceneID   string `gorm:"size:200" json:"scene_id"` // set once the draft has been saved
}

// Scene decodes the proposed scene.
func (d *DraftScene) Scene() (ScrapedScene, error) {
	var s ScrapedScene
	err := json.Unmarshal([]byte(d.SceneJSON), &s)
	return s, err
}

// SetScene encodes the proposed scene.
func (d *DraftScene) SetScene(s ScrapedScene) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	d.SceneJSON = string(b)
	return nil
}

func (d *DraftScene) Save() error {
	db, _ := GetDB()
	defer db.Close()
	return RetryIfBusy(func() error { return db.Save(d).Error })
}

// RetryIfBusy retries fn while SQLite reports the database as locked, which happens while XBVR's
// own scans and scrapes are writing. Unlike SaveWithRetry it returns the final error rather than
// exiting, since a failed draft write should not stop the server.
func RetryIfBusy(fn func() error) error {
	return retry.Do(fn,
		retry.Attempts(12),
		retry.Delay(250*time.Millisecond),
		retry.MaxDelay(3*time.Second),
		retry.LastErrorOnly(true),
		retry.RetryIf(func(err error) bool {
			msg := strings.ToLower(err.Error())
			return strings.Contains(msg, "database is locked") || strings.Contains(msg, "busy")
		}),
	)
}

// FileMatchContext holds per-file matching state. Context is free text the user attaches to a
// file to help match it: studio, performers, a better title, anything the filename lacks. It is
// added to the match screen's search, to web searches, and to what the LLM is told about the file.
// LastSearchedAt records when the LLM scraper last searched the web for the file, so a batch run
// does not spend searches again on files that produced nothing.
type FileMatchContext struct {
	ID             uint       `gorm:"primary_key" json:"id"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FileID         uint       `gorm:"unique_index" json:"file_id"`
	Context        string     `sql:"type:text" json:"context"`
	LastSearchedAt *time.Time `json:"last_searched_at"`
}

// GetFileMatchContext returns the saved context for a file, or an empty string.
func GetFileMatchContext(fileID uint) string {
	db, _ := GetDB()
	defer db.Close()
	var c FileMatchContext
	if db.Where(&FileMatchContext{FileID: fileID}).First(&c).Error != nil {
		return ""
	}
	return c.Context
}

// SetFileMatchContext saves the context for a file.
func SetFileMatchContext(fileID uint, context string) error {
	db, _ := GetDB()
	defer db.Close()
	var c FileMatchContext
	db.Where(&FileMatchContext{FileID: fileID}).FirstOrInit(&c)
	c.FileID = fileID
	c.Context = context
	return RetryIfBusy(func() error { return db.Save(&c).Error })
}

// MarkFileSearched records that the LLM scraper searched the web for a file.
func MarkFileSearched(fileID uint) error {
	db, _ := GetDB()
	defer db.Close()
	var c FileMatchContext
	db.Where(&FileMatchContext{FileID: fileID}).FirstOrInit(&c)
	now := time.Now()
	c.FileID = fileID
	c.LastSearchedAt = &now
	return RetryIfBusy(func() error { return db.Save(&c).Error })
}
