package dms

import (
	"time"

	"gopkg.in/resty.v1"
)

type XbaseBase struct {
	Sites        []string      `json:"sites"`
	Actors       []string      `json:"actors"`
	Tags         []string      `json:"tags"`
	ReleaseGroup []string      `json:"release_group"`
	Volumes      []XbaseVolume `json:"volumes"`
}

type XbaseFile struct {
	ID           int         `json:"ID"`
	CreatedAt    time.Time   `json:"CreatedAt"`
	UpdatedAt    time.Time   `json:"UpdatedAt"`
	DeletedAt    interface{} `json:"DeletedAt"`
	Path         string      `json:"path"`
	Filename     string      `json:"filename"`
	Size         int64       `json:"size"`
	CreatedTime  time.Time   `json:"created_time"`
	UpdatedTime  time.Time   `json:"updated_time"`
	VideoWidth   int         `json:"video_width"`
	VideoHeight  int         `json:"video_height"`
	VideoBitrate uint        `json:"video_bitrate"`
}

type XbaseScenes struct {
	Results int          `json:"results"`
	Scenes  []XbaseScene `json:"scenes"`
}

type XbaseScene struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	SceneID   string    `json:"scene_id"`
	Title     string    `json:"title"`
	SceneType string    `json:"scene_type"`
	Studio    string    `json:"studio"`
	Site      string    `json:"site"`
	Tags      []struct {
		ID     int         `json:"id"`
		Scenes interface{} `json:"scenes"`
		Name   string      `json:"name"`
		Clean  string      `json:"clean"`
		Count  int         `json:"count"`
	} `json:"tags"`
	Cast []struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"cast"`
	Filename []struct {
		ID     int         `json:"id"`
		Name   string      `json:"name"`
		Scenes interface{} `json:"scenes"`
	} `json:"filename"`
	Images []struct {
		URL         string `json:"url"`
		Type        string `json:"type"`
		Orientation string `json:"orientation"`
	} `json:"images"`
	File            []XbaseFile `json:"file"`
	Duration        int         `json:"duration"`
	Synopsis        string      `json:"synopsis"`
	ReleaseDate     time.Time   `json:"release_date"`
	ReleaseDateText string      `json:"release_date_text"`
	CoverURL        string      `json:"cover_url"`
	SceneURL        string      `json:"scene_url"`
	Rating          int         `json:"rating"`
	Favourite       bool        `json:"favourite"`
	Watchlist       bool        `json:"watchlist"`
	IsAvailable     bool        `json:"is_available"`
	IsAccessible    bool        `json:"is_accessible"`
}

type XbaseVolume struct {
	Path string `json:"Path"`
}

func XbaseGet() XbaseBase {
	var data XbaseBase
	resty.R().SetResult(&data).Get("http://127.0.0.1:9999/api/dms/base")
	return data
}

func XbaseGetScene(id string) XbaseScene {
	var data XbaseScene
	resty.R().SetResult(&data).Get("http://127.0.0.1:9999/api/dms/scene?id=" + id)
	return data
}

func XbaseGetFile(id string) XbaseFile {
	var data XbaseFile
	resty.R().SetResult(&data).Get("http://127.0.0.1:9999/api/dms/file?id=" + id)
	return data
}
