package models

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
)

type File struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"created_at" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"updated_at" xbvrbackup:"-"`

	VolumeID    uint      `json:"volume_id" xbvrbackup:"-"`
	Volume      Volume    `json:"-" xbvrbackup:"-"`
	Path        string    `json:"path" xbvrbackup:"path"`
	Filename    string    `json:"filename" xbvrbackup:"filename"`
	Size        int64     `json:"size" xbvrbackup:"size"`
	OsHash      string    `json:"oshash" xbvrbackup:"oshash"`
	CreatedTime time.Time `json:"created_time" xbvrbackup:"created_time"`
	UpdatedTime time.Time `json:"updated_time" xbvrbackup:"updated_time"`

	Type    string `json:"type" xbvrbackup:"type"`
	SceneID uint   `gorm:"index" json:"scene_id" xbvrbackup:"-"`
	Scene   Scene  `json:"-" xbvrbackup:"-"`

	VideoWidth           int     `json:"video_width" xbvrbackup:"video_width"`
	VideoHeight          int     `json:"video_height" xbvrbackup:"video_height"`
	VideoBitRate         int     `json:"video_bitrate" xbvrbackup:"video_bitrate"`
	VideoAvgFrameRate    string  `json:"-" xbvrbackup:"video_avgfps"`
	VideoAvgFrameRateVal float64 `json:"video_avgfps_val" xbvrbackup:"video_avgfps_val"`
	VideoCodecName       string  `json:"video_codec_name" xbvrbackup:"video_codec_name"`
	VideoDuration        float64 `json:"duration" xbvrbackup:"duration"`
	VideoProjection      string  `json:"projection" xbvrbackup:"projection"`
	HasAlpha             bool    `json:"has_alpha" xbvrbackup:"has_alpha"`

	HasHeatmap          bool `json:"has_heatmap" xbvrbackup:"-"`
	IsSelectedScript    bool `json:"is_selected_script" xbvrbackup:"is_selected_script"`
	IsExported          bool `json:"is_exported" xbvrbackup:"-"`
	RefreshHeatmapCache bool `json:"refresh_heatmap_cache" xbvrbackup:"-"`
}

// MarshalJSON customizes the JSON output for File
func (f File) MarshalJSON() ([]byte, error) {
	type Alias File

	// Create a copy of the file
	fileCopy := &struct {
		Alias
		Path string `json:"path"`
	}{
		Alias: Alias(f),
		Path:  f.Path,
	}

	// For Debrid-Link files, modify the path to hide the file ID
	if strings.Contains(f.Path, "||") {
		parts := strings.Split(f.Path, "||")
		fileCopy.Path = parts[0]
	}

	return json.Marshal(fileCopy)
}

func (f *File) GetPath() string {
	if f.Volume.Type == "debridlink" {
		// Extract the file ID stored in f.Path (format: "displayPath||fileID")
		fileID := f.Path
		if strings.Contains(f.Path, "||") {
			parts := strings.Split(f.Path, "||")
			if len(parts) > 1 {
				fileID = parts[1]
			}
		}

		// Create HTTP client and request the file list from Debrid-Link API
		client := &http.Client{}
		httpReq, err := http.NewRequest("GET", "https://debrid-link.com/api/v2/seedbox/list", nil)
		if err != nil {
			return ""
		}
		httpReq.Header.Add("Authorization", "Bearer "+f.Volume.Metadata)

		httpResp, err := client.Do(httpReq)
		if err != nil {
			return ""
		}
		defer httpResp.Body.Close()

		var filesResponse struct {
			Success bool `json:"success"`
			Value   []struct {
				Files []struct {
					ID          string `json:"id"`
					DownloadURL string `json:"downloadUrl"`
				} `json:"files"`
			} `json:"value"`
		}

		if err := json.NewDecoder(httpResp.Body).Decode(&filesResponse); err != nil {
			return ""
		}

		for _, torrent := range filesResponse.Value {
			for _, file := range torrent.Files {
				if file.ID == fileID {
					return file.DownloadURL
				}
			}
		}
		return ""
	}

	return filepath.Join(f.Path, f.Filename)
}

// GetDisplayPath returns the path to be displayed in the UI
func (f *File) GetDisplayPath() string {
	// For Debrid-Link files, only show the display part of the path (before "||")
	if f.Volume.Type == "debridlink" && strings.Contains(f.Path, "||") {
		parts := strings.Split(f.Path, "||")
		return parts[0]
	}

	// For other file types, return the full path
	return f.Path
}

func (f *File) Save() error {
	db, _ := GetDB()
	defer db.Close()

	var err error = retry.Do(
		func() error {
			err := db.Save(&f).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

func (f *File) GetIfExistByPK(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&File{ID: id}).First(f).Error
}

func (f *File) Exists() bool {
	switch f.Volume.Type {
	case "local":
		if _, err := os.Stat(f.GetPath()); os.IsNotExist(err) {
			return false
		}
		return true
	case "putio":
		return true
	case "debridlink":
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
