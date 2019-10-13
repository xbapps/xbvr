package models

import (
	"os"
	"path/filepath"
	"time"
)

type File struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"created_at" json:"-"`
	UpdatedAt time.Time `json:"updated_at" json:"-"`

	VolumeID    uint      `json:"volume_id"`
	Path        string    `json:"path"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	CreatedTime time.Time `json:"created_time"`
	UpdatedTime time.Time `json:"updated_time"`
	SceneID     uint      `json:"-"`
	Scene       Scene     `json:"-"`

	VideoWidth        int     `json:"video_width"`
	VideoHeight       int     `json:"video_height"`
	VideoBitRate      int     `json:"video_bitrate"`
	VideoAvgFrameRate string  `json:"-"`
	VideoCodecName    string  `json:"-"`
	VideoDuration     float64 `json:"duration"`
	VideoProjection   string  `json:"projection"`
}

func (f *File) GetPath() string {
	return filepath.Join(f.Path, f.Filename)
}

func (f *File) Save() error {
	db, _ := GetDB()
	err := db.Save(f).Error
	db.Close()
	return err
}

func (f *File) Exists() bool {
	if _, err := os.Stat(f.GetPath()); os.IsNotExist(err) {
		return false
	}
	return true
}
