package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/transcode"
)

// ResponseTranscodeFFmpeg tells the settings page about the jellyfin-ffmpeg live transcoding runs.
type ResponseTranscodeFFmpeg struct {
	InUse    string                 `json:"in_use"`
	Jellyfin transcode.FFmpegStatus `json:"jellyfin"`
}

// ffmpegRetryInterval is how long a failed jellyfin-ffmpeg download waits before a player's
// request starts another.
const ffmpegRetryInterval = 10 * time.Minute

var ffmpegDownload struct {
	sync.Mutex
	lastStart time.Time
}

// liveFFmpeg returns the jellyfin-ffmpeg that live transcoding runs. While it is not installed
// no streams are offered, and a download is started in the background.
func liveFFmpeg() (string, bool) {
	if path, ok := transcode.ManagedFFmpeg(common.BinDir); ok {
		return path, true
	}
	ffmpegDownload.Lock()
	due := time.Since(ffmpegDownload.lastStart) >= ffmpegRetryInterval
	ffmpegDownload.Unlock()
	if due && !transcode.ManagedFFmpegDownloading() {
		go prepareTranscodeFFmpeg()
	}
	return "", false
}

// prepareTranscodeFFmpeg downloads jellyfin-ffmpeg when it is missing, then finds its encoders.
func prepareTranscodeFFmpeg() {
	ffmpegDownload.Lock()
	ffmpegDownload.lastStart = time.Now()
	ffmpegDownload.Unlock()
	path, err := transcode.EnsureManagedFFmpeg(common.BinDir)
	if err != nil {
		log.Warnf("transcode: live transcoding is off until jellyfin-ffmpeg is installed: %v", err)
		return
	}
	transcode.Reprobe(path)
}

func (i ConfigResource) getTranscodeFFmpeg(req *restful.Request, resp *restful.Response) {
	inUse, _ := transcode.ManagedFFmpeg(common.BinDir)
	resp.WriteHeaderAndEntity(http.StatusOK, ResponseTranscodeFFmpeg{
		InUse:    inUse,
		Jellyfin: transcode.ManagedFFmpegStatus(common.BinDir),
	})
}
