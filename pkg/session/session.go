package session

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/posthog/posthog-go"
	"github.com/xbapps/xbvr/pkg/analytics"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var (
	sessionSource      string
	currentFileID      int
	lastSessionID      uint
	lastSessionSceneID uint
	lastSessionStart   time.Time
	lastSessionEnd     time.Time
)

var currentSessionHeatmap []int

func TrackSessionFromFile(f models.File, doNotTrack string) {
	sessionSource = "file"

	if f.SceneID != 0 && doNotTrack != "true" {
		if lastSessionSceneID != f.SceneID {
			if lastSessionID != 0 {
				watchSessionFlush()
			}

			lastSessionSceneID = f.SceneID
			lastSessionStart = time.Now()
			newWatchSession()
		}

		lastSessionEnd = time.Now()
	}
}

func FinishTrackingFromFile(doNotTrack string) {
	lastSessionEnd = time.Now()
	if doNotTrack == "false" {
		watchSessionFlush()
	}
}

func TrackSessionFromRemote(packet DeoPacket) {
	if packet.Path == "" || packet.Duration == 0 {
		return
	}

	sessionSource = "deovr"

	tmpPath, err := url.Parse(packet.Path)
	if err != nil {
		return
	}
	tmpCurrentFileID, err := strconv.Atoi(path.Base(tmpPath.Path))
	if err != nil {
		return
	}

	// Currently playing file has changed
	if tmpCurrentFileID != currentFileID {
		// Get scene ID
		currentFileID = tmpCurrentFileID

		f := models.File{}
		db, _ := models.GetDB()
		err = db.First(&f, currentFileID).Error
		defer db.Close()

		// Flush old session
		if lastSessionID != 0 {
			watchSessionFlush()
		}

		// Create new session
		lastSessionSceneID = f.SceneID
		lastSessionStart = time.Now()
		newWatchSession()

		currentSessionHeatmap = make([]int, int(packet.Duration))
	}

	// Keep session alive if Deo is playing
	if packet.PlayerState == PLAYING {
		lastSessionEnd = time.Now()

		position := int(packet.CurrentTime)
		if position != 0 && len(currentSessionHeatmap) >= position {
			currentSessionHeatmap[position] = currentSessionHeatmap[position] + 1
		}
	}
}

func CheckForDeadSession() {
	var timeout float64
	if sessionSource == "file" {
		timeout = 60
	} else {
		timeout = 5
	}

	if time.Since(lastSessionEnd).Seconds() > timeout && lastSessionSceneID != 0 && lastSessionID != 0 {
		watchSessionFlush()
		lastSessionID = 0
		lastSessionSceneID = 0
	}
}

func newWatchSession() {
	obj := models.History{SceneID: lastSessionSceneID, TimeStart: lastSessionStart}
	obj.Save()

	var scene models.Scene
	err := scene.GetIfExistByPK(lastSessionSceneID)
	if err == nil {
		scene.LastOpened = time.Now()
		scene.Save()
	}

	lastSessionID = obj.ID

	analytics.Event("watchsession-new", posthog.NewProperties().Set("scene-id", scene.SceneID))
	common.Log.Infof("New session #%v for scene #%v from %v", lastSessionID, lastSessionSceneID, sessionSource)
}

func watchSessionFlush() {
	var obj models.History
	err := obj.GetIfExist(lastSessionID)
	if err == nil {
		obj.TimeEnd = lastSessionEnd
		obj.Duration = time.Since(lastSessionStart).Seconds()
		obj.Save()

		var scene models.Scene
		err := scene.GetIfExistByPK(lastSessionSceneID)
		if err == nil {
			if !scene.IsWatched {
				scene.IsWatched = true
				scene.Save()
			}
		}

		common.Log.Infof("Session #%v duration for scene #%v is %v", lastSessionID, lastSessionSceneID, time.Since(lastSessionStart).Seconds())

		// Dump heatmap
		// TODO: handle multipart scenes
		if !scene.IsMultipart && sessionSource == "deovr" {
			path := path.Join(common.HeatmapDir, fmt.Sprintf("%v.json", lastSessionSceneID))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				// Create new heatmap
				data, _ := json.Marshal(currentSessionHeatmap)
				ioutil.WriteFile(path, data, 0644)
			} else {
				// Update existing heatmap
				b, err := ioutil.ReadFile(path)
				if err != nil {
					return
				}

				tmpHeatmap := make([]int, len(currentSessionHeatmap))
				err = json.Unmarshal(b, &tmpHeatmap)
				if err != nil {
					return
				}

				for k, v := range tmpHeatmap {
					currentSessionHeatmap[k] = currentSessionHeatmap[k] + v
				}

				data, _ := json.Marshal(currentSessionHeatmap)
				ioutil.WriteFile(path, data, 0644)
			}
		}
	}

	currentFileID = 0
	lastSessionID = 0
	lastSessionSceneID = 0
}
