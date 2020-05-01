package xbvr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/darwayne/go-timecode/timecode"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/vansante/go-ffprobe.v2"
)

func GeneratePreviews() {
	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Model(&models.Scene{}).Where("is_available = ?", true).Where("has_video_preview = ?", false).Order("release_date desc").Find(&scenes)

	for _, scene := range scenes {
		files, _ := scene.GetFiles()
		if len(files) > 0 {
			if files[0].Exists() {
				log.Infof("Rendering %v", scene.SceneID)
				err := renderPreview(scene.SceneID, files[0].GetPath())
				if err == nil {
					scene.HasVideoPreview = true
					scene.Save()
				} else {
					log.Warn(err)
				}
			}
		}
	}
}

func renderPreview(sceneID string, inputFile string) error {
	startTime := 10
	snippetLength := 0.4
	snippetAmount := 20

	tmpPath := filepath.Join(common.VideoPreviewDir, "tmp")
	os.MkdirAll(tmpPath, os.ModePerm)
	defer os.RemoveAll(tmpPath)

	// Get video duration
	ffprobeReader, err := os.Open(inputFile)
	if err != nil {
		return err
	}

	ffdata, err := ffprobe.ProbeReader(context.Background(), ffprobeReader)
	if err != nil {
		return err
	}
	vs := ffdata.FirstVideoStream()
	dur, err := strconv.ParseFloat(vs.Duration, 64)
	if err != nil {
		return err
	}

	// Prepare snippets
	interval := dur/float64(snippetAmount) - float64(startTime)
	for i := 1; i <= snippetAmount; i++ {
		start := time.Duration(float64(i)*interval+float64(startTime)) * time.Second
		snippetFile := filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", i))
		cmd := []string{
			"-y",
			"-ss", strings.TrimSuffix(timecode.New(start, timecode.IdentityRate).String(), ":00"),
			"-i", inputFile,
			"-vf", "crop=in_w/2:in_h:in_w/2:in_h,scale=400:400",
			"-t", fmt.Sprintf("%v",snippetLength),
			"-an", snippetFile,
		}

		err := exec.Command(GetBinPath("ffmpeg"), cmd...).Run()
		if err != nil {
			return err
		}
	}

	// Prepare concat file
	concatFile := filepath.Join(tmpPath, "concat.txt")
	f, err := os.Create(concatFile)
	if err != nil {
		return err
	}
	for i := 1; i <= snippetAmount; i++ {
		f.WriteString(fmt.Sprintf("file '%v.mp4'\n", i))
	}
	f.Close()

	// Save result
	destFile := filepath.Join(common.VideoPreviewDir, sceneID+".mp4")
	cmd := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		"-c", "copy",
		destFile,
	}
	err = exec.Command(GetBinPath("ffmpeg"), cmd...).Run()
	if err != nil {
		return err
	}

	return nil
}
