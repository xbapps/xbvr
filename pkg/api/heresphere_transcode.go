package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/transcode"
)

func transcodeOptions(bitrateMbps int) transcode.Options {
	c := config.Config.Interfaces.Heresphere
	return transcode.Options{
		BitrateMbps: bitrateMbps,
		AudioKbps:   c.TranscodeAudioKbps,
		MaxHeight:   c.TranscodeMaxHeight,
	}
}

// transcodeBitrates returns the video bitrates offered for each codec, lowest first.
func transcodeBitrates() []int {
	if b := transcode.ParseBitrates(config.Config.Interfaces.Heresphere.TranscodeBitrates); len(b) > 0 {
		return b
	}
	return []int{transcode.DefaultBitrateMbps}
}

// transcodeBitrate returns the bitrate a stream URL asks for: one of the offered bitrates,
// or the lowest when the URL names none.
func transcodeBitrate(param string) (int, bool) {
	offered := transcodeBitrates()
	param = strings.TrimSpace(param)
	if param == "" {
		return offered[0], true
	}
	mbps, err := strconv.Atoi(param)
	if err != nil {
		return 0, false
	}
	for _, b := range offered {
		if b == mbps {
			return mbps, true
		}
	}
	return 0, false
}

// PrimeTranscodeHardware gets ffmpeg ready in the background at startup, so the first scene
// a player opens does not wait for the download or the probe.
func PrimeTranscodeHardware() {
	if config.Config.Interfaces.Heresphere.TranscodeEnabled {
		go prepareTranscodeFFmpeg()
	}
}

// transcodeStreamName names a live stream in the player's list, e.g. "H264 20 Mbps (hwe)" for
// a hardware encode or "x264 20 Mbps (swe)" for a software encode on the CPU. prefix (the
// file's position) is added when the scene has more than one file.
func transcodeStreamName(prefix, codec string, bitrateMbps int, software bool) string {
	name := fmt.Sprintf("%s %d Mbps", transcode.StreamName(codec), bitrateMbps)
	if software {
		name += " (swe)"
	} else {
		name += " (hwe)"
	}
	if prefix != "" {
		name = prefix + " " + name
	}
	return name
}

// heresphereTranscodeMedia describes the live transcode streams as media entries, one per
// enabled codec and offered bitrate. A live stream has no file, so its size is the estimated
// byte count at its bitrates. prefix is empty for a scene with one file.
func heresphereTranscodeMedia(req *restful.Request, file models.File, prefix, dnt string) []HeresphereMedia {
	c := config.Config.Interfaces.Heresphere
	if !c.TranscodeEnabled || file.VideoDuration <= 0 || transcodeFFmpegPending() {
		return nil
	}
	codecs := transcode.ParseCodecs(c.TranscodeCodecs)
	if len(codecs) == 0 {
		codecs = []string{transcode.CodecH264}
	}
	bitrates := transcodeBitrates()
	hw := transcode.Detect(ffmpegBin())

	var media []HeresphereMedia
	seenEncoder := map[string]bool{}
	for _, codec := range codecs {
		encoder := ""
		for _, bitrate := range bitrates {
			opts := transcodeOptions(bitrate)
			size := opts.VirtualSize(file.VideoDuration)
			if size <= 0 {
				continue
			}
			plan := transcode.PlanForEncoder(codec, hw, opts, file.VideoWidth, file.VideoHeight, file.VideoAvgFrameRateVal)
			if plan.Unavailable || plan.Encoder == "" {
				continue
			}
			// Two codec choices can land on the same encoder, e.g. H.264 without hardware and
			// x264; offer that encoder once.
			if encoder == "" {
				if seenEncoder[plan.Encoder] {
					break
				}
				encoder = plan.Encoder
				seenEncoder[encoder] = true
			}
			height, width := plan.Height, plan.Width
			if height <= 0 {
				height = file.VideoHeight
			}
			if width <= 0 {
				width = file.VideoWidth
			}
			resolution := strconv.Itoa(height)
			if file.VideoProjection == "360_tb" {
				resolution = strconv.Itoa(height / 2)
			}
			params := "?codec=" + url.QueryEscape(codec) + "&bitrate=" + strconv.Itoa(bitrate)
			if dnt != "" {
				params += "&dnt=true"
			}
			if isDeoAuthEnabled() {
				params += "&token=" + transcodeToken(file.ID, codec)
			}
			media = append(media, HeresphereMedia{
				Name: transcodeStreamName(prefix, codec, bitrate, plan.Backend.Name == transcode.BackendCPU),
				Sources: []HeresphereSource{{
					Resolution: StringOrInt(resolution),
					Height:     height,
					Width:      width,
					Size:       size,
					URL:        fmt.Sprintf("%v://%v/api/dms/transcode/%v%v", getProto(req), req.Request.Host, file.ID, params),
				}},
			})
		}
	}
	return media
}
