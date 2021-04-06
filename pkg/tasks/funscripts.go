package tasks

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/xbapps/xbvr/pkg/models"
)

func ExportFunscripts(w http.ResponseWriter) {

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"funscripts.zip\"")

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

		for _, file := range scriptFiles {
			if file.Exists() {
				funscriptName := fmt.Sprintf("%s.funscript", scene.GetFunscriptTitle())
				log.Infof("adding " + funscriptName)

				if err = AddFileToZip(zipWriter, file.GetPath(), funscriptName); err != nil {
					log.Infof("Error when adding file to zip: %v", err)
					continue
				}
				break
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
