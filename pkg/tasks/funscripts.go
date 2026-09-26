package tasks

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/models"
)

func ExportFunscripts(w http.ResponseWriter, updatedOnly bool) {

	w.Header().Set("Content-Type", "application/zip")
	if updatedOnly {
		w.Header().Set("Content-Disposition", "attachment; filename=\"funscripts-update.zip\"")
	} else {
		w.Header().Set("Content-Disposition", "attachment; filename=\"funscripts.zip\"")
	}

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Model(&models.Scene{}).Where("is_scripted = ?", true).Order("scene_id").Find(&scenes)

	for _, scene := range scenes {
		scriptFiles, err := scene.GetScriptFiles()
		if err != nil {
			log.Error(err)
			return
		}

		for i, file := range scriptFiles {
			if i == 0 {
				if file.Exists() {
					if !file.IsExported || !updatedOnly {
						funscriptName := fmt.Sprintf("%s.funscript", scene.GetFunscriptTitle())

						if err = AddFileToZip(zipWriter, file.GetPath(), funscriptName); err != nil {
							log.Infof("Error when adding file to zip: %v (%s)", err, funscriptName)
							continue
						}
					}

					if !file.IsExported {
						file.IsExported = true
						file.Save()
					}
				}
			} else {
				if file.IsExported {
					file.IsExported = false
					file.Save()
				}
			}
		}
	}
}

func AddFileToZip(zipWriter *zip.Writer, srcfilename, zipfilename string) error {

	fileToZip, err := os.Open(srcfilename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = zipfilename

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

func GenerateFunscriptSpeeds(tlog *logrus.Entry) {
	if !models.CheckLock("funscript_speeds") {
		models.CreateLock("funscript_speeds")

		db, _ := models.GetDB()
		defer db.Close()

		var scriptfiles []models.File
		db.Model(&models.File{}).Preload("Volume").Where("type = ?", "script").Where("funscript_speed = ?", 0).Find(&scriptfiles)

		for i, file := range scriptfiles {
			if tlog != nil && (i%50) == 0 {
				tlog.Infof("Generating funscript speeds (%v/%v)", i+1, len(scriptfiles))
			}
			if file.Exists() {
				speed, err := CalculateFunscriptSpeed(file.GetPath())

				if err == nil {
					file.FunscriptSpeed = speed
					file.Save()
				} else {
					log.Warn(err)
				}
			}
		}
	}

	models.RemoveLock("funscript_speeds")
}

func CalculateFunscriptSpeed(inputFile string) (int, error) {
	funscript, err := LoadFunscriptData(inputFile)
	if err != nil {
		return 0, err
	}
	funscript.UpdateSpeed()
	return funscript.CalculateMedian(), nil
}

func (funscript *Script) UpdateSpeed() {
	var t1, t2 int64
	var p1, p2 int

	for i := range funscript.Actions {
		if i == 0 {
			continue
		}
		t1 = funscript.Actions[i].At
		t2 = funscript.Actions[i-1].At
		p1 = funscript.Actions[i].Pos
		p2 = funscript.Actions[i-1].Pos

		speed := math.Abs(float64(p1-p2)) / float64(t1-t2) * 1000
		funscript.Actions[i].Speed = speed
	}
}

func (funscript *Script) CalculateMedian() int {
	sort.Slice(funscript.Actions, func(i, j int) bool {
		return funscript.Actions[i].Speed < funscript.Actions[j].Speed
	})

	mNumber := len(funscript.Actions) / 2

	if len(funscript.Actions)%2 != 0 {
		return int(funscript.Actions[mNumber].Speed)
	}

	return int((funscript.Actions[mNumber-1].Speed + funscript.Actions[mNumber].Speed) / 2)
}
