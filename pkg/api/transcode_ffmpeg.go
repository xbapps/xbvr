package api

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/transcode"
)

// ResponseTranscodeFFmpeg tells the settings page which ffmpeg live transcoding uses.
type ResponseTranscodeFFmpeg struct {
	Source   string                 `json:"source"`
	InUse    string                 `json:"in_use"`
	Jellyfin transcode.FFmpegStatus `json:"jellyfin"`
}

// useJellyfinFFmpeg reports whether live transcoding uses the downloaded jellyfin-ffmpeg.
func useJellyfinFFmpeg() bool {
	return config.Config.Interfaces.Heresphere.TranscodeFFmpeg != config.TranscodeFFmpegSystem
}

// transcodeFFmpegPending reports whether live transcoding is waiting for jellyfin-ffmpeg to
// download. No streams are offered until it finishes, so players do not see a list from the
// system ffmpeg that changes a minute later.
func transcodeFFmpegPending() bool {
	return useJellyfinFFmpeg() && transcode.ManagedFFmpegDownloading()
}

// prepareTranscodeFFmpeg downloads jellyfin-ffmpeg when it is selected and missing, then
// finds the encoders of the ffmpeg in use. If the download fails, the system ffmpeg is used.
func prepareTranscodeFFmpeg() {
	if useJellyfinFFmpeg() {
		if _, err := transcode.EnsureManagedFFmpeg(common.BinDir); err != nil {
			log.Warnf("transcode: using %s because jellyfin-ffmpeg is not available: %v", systemFFmpegBin(), err)
		}
	}
	transcode.Reprobe(ffmpegBin())
}

func (i ConfigResource) getTranscodeFFmpeg(req *restful.Request, resp *restful.Response) {
	source := config.TranscodeFFmpegJellyfin
	if !useJellyfinFFmpeg() {
		source = config.TranscodeFFmpegSystem
	}
	inUse := ffmpegBin()
	if transcodeFFmpegPending() {
		inUse = ""
	}
	resp.WriteHeaderAndEntity(http.StatusOK, ResponseTranscodeFFmpeg{
		Source:   source,
		InUse:    inUse,
		Jellyfin: transcode.ManagedFFmpegStatus(common.BinDir),
	})
}
