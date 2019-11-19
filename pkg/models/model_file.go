package models

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type File struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"created_at" json:"-"`
	UpdatedAt time.Time `json:"updated_at" json:"-"`

	VolumeID    uint      `json:"volume_id"`
	Volume      Volume    `json:"-"`
	Path        string    `json:"path"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	CreatedTime time.Time `json:"created_time"`
	UpdatedTime time.Time `json:"updated_time"`

	SceneID uint  `json:"scene_id"`
	Scene   Scene `json:"-"`

	VideoWidth           int     `json:"video_width"`
	VideoHeight          int     `json:"video_height"`
	VideoBitRate         int     `json:"video_bitrate"`
	VideoAvgFrameRate    string  `json:"-"`
	VideoAvgFrameRateVal float64 `json:"video_avgfps_val"`
	VideoCodecName       string  `json:"-"`
	VideoDuration        float64 `json:"duration"`
	VideoProjection      string  `json:"projection"`
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
	switch f.Volume.Type {
	case "local":
		if _, err := os.Stat(f.GetPath()); os.IsNotExist(err) {
			return false
		}
		return true
	case "putio":
		// NOTE: we're assuming files weren't removed via Put.io web UI, so there's no need to check
		return true
	default:
		return false
	}
}

func (f *File) CalculateFramerate() error {
	v1, err := strconv.ParseFloat(strings.Split(f.VideoAvgFrameRate, "/")[0], 64)
	if err != nil {
		return err
	}

	v2, err := strconv.ParseFloat(strings.Split(f.VideoAvgFrameRate, "/")[1], 64)
	if err != nil {
		return err
	}

	f.VideoAvgFrameRateVal = math.Ceil(v1 / v2)
	return nil
}
