package xbase

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/creasty/defaults"
	"github.com/djherbis/times"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"github.com/vansante/go-ffprobe"
	"gopkg.in/cheggaaa/pb.v1"
)

type Volume struct {
	gorm.Model
	Path        string
	LastScan    time.Time
	IsEnabled   bool
	IsAvailable bool
}

func (o *Volume) IsMounted() bool {
	if _, err := os.Stat(o.Path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (o *Volume) Save() error {
	db, _ := GetDB()
	err := db.Save(o).Error
	db.Close()
	return err
}

func (o *Volume) Files() []File {
	var allFiles []File
	db, _ := GetDB()
	db.Where("path LIKE ?", o.Path+"%").Find(&allFiles)
	db.Close()
	return allFiles
}

func (o *Volume) Rescan() error {
	if o.IsMounted() {
		notAllowedFn := []string{".DS_Store", ".tmp"}
		allowedExt := []string{".mp4", ".avi", ".wmv", ".mpeg4", ".mov"}

		procList := make([]string, 0)

		_ = filepath.Walk(o.Path, func(path string, f os.FileInfo, err error) error {
			if !f.Mode().IsDir() {
				// Make sure the filename should be considered
				if !funk.Contains(notAllowedFn, filepath.Base(path)) && funk.Contains(allowedExt, strings.ToLower(filepath.Ext(path))) {

					// cleanPath := strings.Replace(path, o.Path+string(os.PathSeparator), "", -1)

					var fl File
					db, _ := GetDB()
					err = db.Where(&File{Path: filepath.Dir(path), Filename: filepath.Base(path)}).First(&fl).Error
					db.Close()

					if err == gorm.ErrRecordNotFound {
						procList = append(procList, path)
					}
				}
			}
			return nil
		})

		bar := pb.StartNew(len(procList))
		for _, path := range procList {
			fStat, _ := os.Stat(path)
			fTimes, _ := times.Stat(path)
			// fHash, _ := hashFileXX(path)

			var fl File
			fl = File{
				Path:        filepath.Dir(path),
				Filename:    filepath.Base(path),
				Size:        fStat.Size(),
				CreatedTime: fTimes.BirthTime(),
				UpdatedTime: fTimes.ModTime(),
				// Hash:        fHash,
			}

			ffdata, err := ffprobe.GetProbeData(path, time.Second*3)
			if err != nil {
				log.Errorf("Error running ffprobe", path, err)
			}

			vs := ffdata.GetFirstVideoStream()
			bitRate, _ := strconv.Atoi(vs.BitRate)
			fl.VideoAvgFrameRate = vs.AvgFrameRate
			fl.VideoBitRate = bitRate
			fl.VideoCodecName = vs.CodecName
			fl.VideoWidth = vs.Width
			fl.VideoHeight = vs.Height

			err = fl.Save()
			if err != nil {
				log.Errorf("New file %s, but got error %s", path, err)
			}

			bar.Increment()
		}

		bar.Finish()

		o.LastScan = time.Now()
		o.Save()

		// Check if files are still present at the location
		allFiles := o.Files()
		for i := range allFiles {
			if !allFiles[i].Exists() {
				log.Info(allFiles[i].GetPath())
				db, _ := GetDB()
				db.Delete(&allFiles[i])
				db.Close()
			}
		}
	}

	return nil
}

func (o *Volume) SaveLocalInfo() {
	if o.IsMounted() {
		var files []File

		db, _ := GetDB()
		_ = db.Where("path LIKE ?", o.Path+"%").Find(&files).Error
		defer db.Close()

		for i := range files {
			fn := files[i].Filename

			var pfn PossibleFilename
			var scenes []Scene
			db.Where(&PossibleFilename{Name: path.Base(fn)}).First(&pfn)
			db.Model(&pfn).Preload("Cast").Preload("Tags").Related(&scenes, "Scenes")

			if len(scenes) == 1 {
				downloadFile(scenes[0].CoverURL, files[i].GetPath()+".png")
				saveJSON(scenes[0], files[i].GetPath()+".json")

				files[i].SceneID = scenes[0].ID
				files[i].Save()
			}
		}
	}
}

func caseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get("http://127.0.0.1:9999/img/700x/" + strings.Replace(url, "://", ":/", -1))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("HTTP status code %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

type InfoFile struct {
	Position         float32 `json:"position" default:"0.0"`
	DisplayName      string  `json:"display"`
	PresetID         int     `json:"presetId" default:"0"`
	RememberPosition int     `json:"rememberPosition" default:"1"`
	Present          bool    `json:"present" default:"false"`
	PlaybackType     int     `json:"playbackType" default:"15"`
	Type             int     `json:"type" default:"1"`
	SceneDetails     Scene   `json:"scene_details"`
}

func saveJSON(sc Scene, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	info := &InfoFile{}
	defaults.Set(info)

	c := make([]string, 0)
	for i := range sc.Cast {
		c = append(c, sc.Cast[i].Name)
	}

	info.DisplayName = strings.Join(c, ", ") + " - " + sc.Title
	info.SceneDetails = sc

	infoJSON, _ := json.Marshal(info)

	out.Write([]byte(infoJSON))
	if err != nil {
		return err
	}

	return nil
}

func CheckVolumes() {
	db, _ := GetDB()
	defer db.Close()

	var vol []Volume
	db.Find(&vol)

	for i := range vol {
		vol[i].IsAvailable = vol[i].IsMounted()
		vol[i].Save()
	}
}
