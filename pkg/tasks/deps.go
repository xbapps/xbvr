package tasks

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-resty/resty/v2"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/ffprobe"
)

func CheckDependencies() {
	// Check ffprobe
	ffprobePath := filepath.Join(common.BinDir, "ffprobe")
	if runtime.GOOS == "windows" {
		ffprobePath = ffprobePath + ".exe"
	}
	if _, err := os.Stat(ffprobePath); os.IsNotExist(err) {
		log.Info("ffprobe not installed, downloading now...")
		downloadFfbinaries("ffprobe")
	}

	// Check ffmpeg
	ffmpegPath := filepath.Join(common.BinDir, "ffmpeg")
	if runtime.GOOS == "windows" {
		ffmpegPath = ffmpegPath + ".exe"
	}
	if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
		log.Info("ffmpeg not installed, downloading now...")
		downloadFfbinaries("ffmpeg")
	}

	// Set path for go-ffprobe
	ffprobe.SetFFProbeBinPath(ffprobePath)
}

func GetBinPath(tool string) string {
	path := filepath.Join(common.BinDir, tool)
	if runtime.GOOS == "windows" {
		path = path + ".exe"
	}
	return path
}

func downloadFfbinaries(tool string) error {
	var platformId = ""
	if runtime.GOOS == "windows" {
		switch runtime.GOARCH {
		case "386":
			platformId = "windows-32"
		default:
			platformId = "windows-64"
		}
	}
	if runtime.GOOS == "darwin" {
		platformId = "osx-64"
	}
	if runtime.GOOS == "linux" {
		switch runtime.GOARCH {
		case "386":
			platformId = "linux-32"
		case "amd64":
			platformId = "linux-64"
		case "arm":
			platformId = "linux-armhf"
		case "arm64":
			platformId = "linux-arm64"
		}
	}

	if platformId == "" {
		return errors.Errorf("Unknown architecture: %v/%v", runtime.GOOS, runtime.GOARCH)
	}

	resp, err := resty.New().R().Get("https://ffbinaries.com/api/v1/version/4.2.1")
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return errors.Errorf("HTTP status code %d", resp.StatusCode())
	}

	url := gjson.Get(resp.String(), "bin."+platformId+"."+tool)

	err = downloadFile(url.String(), filepath.Join(common.BinDir, tool+".zip"))
	if err != nil {
		return err
	}

	err = archiver.Unarchive(filepath.Join(common.BinDir, tool+".zip"), common.BinDir)
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(common.BinDir, tool+".zip"))

	return nil
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
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
