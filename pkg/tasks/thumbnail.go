package tasks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/ffprobe"
	"github.com/xbapps/xbvr/pkg/models"
)

func GenerateThumnbnails(endTime *time.Time) {
	if !models.CheckLock("thumnbnails") {
		models.CreateLock("thumnbnails")
		defer models.RemoveLock("thumnbnails")
		log.Infof("Generating thumnbnails")
		db, _ := models.GetDB()
		defer db.Close()

		var scenes []models.Scene
		db.Model(&models.Scene{}).Where("is_available = ?", true).Where("has_thumbnail = ?", false).Order("release_date desc").Find(&scenes)

		for _, scene := range scenes {
			files, _ := scene.GetFiles()
			if len(files) > 0 {
				if endTime != nil && time.Now().After(*endTime) {
					return
				}
				i := 0
				for i < len(files) && files[i].Exists() {
					if files[i].Type == "video" {
						log.Infof("Thumbnail Rendering %v", scene.SceneID)

						name := filepath.Base(files[i].Filename)
						nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))

						destFile := filepath.Join(common.VideoThumbnailDir, nameWithoutExt+".jpg")
						err := RenderThumnbnails(
							files[i].GetPath(),
							destFile,
							files[i].VideoProjection,
							config.Config.Library.Preview.StartTime,
							config.Config.Library.Preview.SnippetLength,
							config.Config.Library.Preview.SnippetAmount,
							config.Config.Library.Preview.Resolution,
							config.Config.Library.Preview.ExtraSnippet,
						)
						if err == nil {
							scene.HasThumbnail = true
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
	log.Infof("Thumnbnails generated")
}

func RenderThumnbnails(inputFile string, destFile string, videoProjection string, startTime int, snippetLength float64, snippetAmount int, resolution int, extraSnippet bool) error {
	// tmpPath := filepath.Join(common.VideoThumbnailDir, "tmp")
	// os.MkdirAll(tmpPath, os.ModePerm)
	// defer os.RemoveAll(tmpPath)

	os.MkdirAll(common.VideoThumbnailDir, os.ModePerm)

	// Get video duration
	ffdata, err := ffprobe.GetProbeData(inputFile, time.Second*10)
	if err != nil {
		return err
	}
	vs := ffdata.GetFirstVideoStream()
	// dur := ffdata.Format.DurationSeconds

	crop := "iw/2:ih:iw/2:ih" // LR videos
	if vs.Height == vs.Width {
		crop = "iw/2:ih/2:iw/4:ih/2" // TB videos
	}
	if videoProjection == "flat" {
		crop = "iw:ih:iw:ih" // LR videos
	}
	// Mono 360 crop args: (no way of accurately determining)
	// "iw/2:ih:iw/4:ih"
	vfArgs := fmt.Sprintf("crop=%v,scale=%v:-1:flags=lanczos,fps=fps=1/%v:round=down,tile=10x20", crop, resolution, 30)

	args := []string{}
	if isCUDAEnabled() {
		args = []string{
			"-y",
			"-ss", "5",
			"-hwaccel", "cuda",
			"-skip_frame",
			"nokey",
			"-i", inputFile,
			// "-t", "60",
			"-vf", vfArgs,
			// "-frame_pts", "true",
			"-q:v", "3",
			// "-pix_fmt", "rgb24",
			// "-c:v", "mjpeg",
			//"-f", "image2pipe",
			//"-",
			destFile,
		}
	} else {
		args = []string{
			"-y",
			"-ss", "5",
			// "-hwaccel", "cuda",
			"-skip_frame",
			"nokey",
			"-i", inputFile,
			// "-t", "60",
			"-vf", vfArgs,
			// "-frame_pts", "true",
			"-q:v", "3",
			// "-pix_fmt", "rgb24",
			// "-c:v", "mjpeg",
			//"-f", "image2pipe",
			//"-",
			destFile,
		}
	}

	cmd := buildCmd(GetBinPath("ffmpeg"), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Stderr: %s\n", stderr.String()) // Stderrを出力
		return err
	}
	return nil
}

func isCUDAEnabled() bool {
	args := []string{
		"-hide_banner",
		"-hwaccels",
	}
	cmd := buildCmd(GetBinPath("ffmpeg"), args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false // エラーが発生した場合はCUDAが利用できないとみなす
	}

	// コマンドの出力を確認し、CUDAが利用可能かどうかを判定する
	output := out.String()
	return strings.Contains(output, "cuda")
}
