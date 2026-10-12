package models

import "time"

// R18devContent remembers R18.dev content IDs the crawler shouldn't fetch again (rejected) or has
// to look at again soon (pending). A reject is a property of the scene (not VR, compilation,
// physical disc, ...), so every R18.dev Scraper shares it. Bounded by R18.dev's VR catalog.
type R18devContent struct {
	ContentID string    `gorm:"primary_key;size:64" json:"content_id"`
	State     string    `gorm:"size:16;index" json:"state"` // R18devRejected or R18devPending
	Reason    string    `gorm:"size:255" json:"reason"`
	ScraperID string    `gorm:"size:255;index" json:"scraper_id"` // pending: the Scraper that rechecks it
	CheckedAt time.Time `json:"checked_at"`
}

func (R18devContent) TableName() string { return "r18dev_contents" }

const (
	R18devRejected = "rejected"
	R18devPending  = "pending"
)

// R18devSource is the scene count of one R18.dev list (label / studio / category) after the last
// clean run. Scenes never get removed from R18.dev, so an unchanged count means nothing new.
type R18devSource struct {
	Key       string    `gorm:"primary_key;size:64" json:"key"` // "label:25637"
	Total     int       `json:"total"`
	CheckedAt time.Time `json:"checked_at"`
}

func (R18devSource) TableName() string { return "r18dev_sources" }
