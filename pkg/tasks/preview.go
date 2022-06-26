package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/darwayne/go-timecode/timecode"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
)

func GeneratePreviews() {
	if !models.CheckLock("previews") {
		models.CreateLock("previews")

		db, _ := models.GetDB()
		defer db.Close()

		var scenes []models.Scene
		db.Model(&models.Scene{}).Where("is_available = ?", true).Where("has_video_preview = ?", false).Order("release_date desc").Find(&scenes)

		for _, scene := range scenes {
			files, _ := scene.GetFiles()
			if len(files) > 0 {
				i := 0
				for i < len(files) && files[i].Exists() {
					if files[i].Type == "video" {
						log.Infof("Rendering %v", scene.SceneID)
						destFile := filepath.Join(common.VideoPreviewDir, scene.SceneID+".mp4")
						err := RenderPreview(
							files[i].GetPath(),
							destFile,
							config.Config.Library.Preview.StartTime,
							config.Config.Library.Preview.SnippetLength,
							config.Config.Library.Preview.SnippetAmount,
							config.Config.Library.Preview.Resolution,
							config.Config.Library.Preview.ExtraSnippet,
						)
						if err == nil {
							scene.HasVideoPreview = true
							scene.Save()
							break
						} else {
							log.Warn(err)
						}
					}
					i++
				}
			}
		}
	}

	models.RemoveLock("previews")
}

func RenderPreview(inputFile string, destFile string, startTime int, snippetLength float64, snippetAmount int, resolution int, extraSnippet bool) error {
	tmpPath := filepath.Join(common.VideoPreviewDir, "tmp")
	os.MkdirAll(tmpPath, os.ModePerm)
	defer os.RemoveAll(tmpPath)

	// Get video duration
	ffdata, err := ffprobe.GetProbeData(inputFile, time.Second*3)
	if err != nil {
		return err
	}
	vs := ffdata.GetFirstVideoStream()
	dur := ffdata.Format.DurationSeconds

	crop := "iw/2:ih:iw/2:ih" // LR videos
	if vs.Height == vs.Width {
		crop = "iw/2:ih/2:iw/4:ih/2" // TB videos
	}
	// Mono 360 crop args: (no way of accurately determining)
	// "iw/2:ih:iw/4:ih"
	vfArgs := fmt.Sprintf("crop=%v,scale=%v:%v", crop, resolution, resolution)

	// Prepare snippets
	interval := (dur - float64(startTime)) / float64(snippetAmount)
	for i := 1; i <= snippetAmount; i++ {
		start := time.Duration(float64(i)*interval+float64(startTime)) * time.Second
		snippetFile := filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", i))
		cmd := []string{
			"-y",
			"-ss", strings.TrimSuffix(timecode.New(start, timecode.IdentityRate).String(), ":00"),
			"-i", inputFile,
			"-vf", vfArgs,
			"-pix_fmt", "yuv420p",
			"-t", fmt.Sprintf("%v", snippetLength),
			"-an", snippetFile,
		}

		err := exec.Command(GetBinPath("ffmpeg"), cmd...).Run()
		if err != nil {
			return err
		}
	}

	// Ensure ending is always in preview
	if extraSnippet && dur/float64(snippetAmount) > float64(150) {
		snippetAmount = snippetAmount + 1

		start := time.Duration(dur-float64(150)) * time.Second
		snippetFile := filepath.Join(tmpPath, fmt.Sprintf("%v.mp4", snippetAmount))
		cmd := []string{
			"-y",
			"-ss", strings.TrimSuffix(timecode.New(start, timecode.IdentityRate).String(), ":00"),
			"-i", inputFile,
			"-vf", vfArgs,
			"-t", fmt.Sprintf("%v", snippetLength),
			"-an", snippetFile,
		}

		err = exec.Command(GetBinPath("ffmpeg"), cmd...).Run()
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
	cmd := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", filepath.ToSlash(concatFile),
		"-c", "copy",
		filepath.ToSlash(destFile),
	}
	err = exec.Command(GetBinPath("ffmpeg"), cmd...).Run()
	if err != nil {
		return err
	}

	return nil
}
