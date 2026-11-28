package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/transcode"
)

type RequestTestTranscode struct {
	Codecs         []string `json:"codecs"`
	SampleLimit    int      `json:"sample_limit"`
	DurationSecond float64  `json:"duration_seconds"`
}

type TestTranscodeSample struct {
	FileID   uint    `json:"file_id"`
	Name     string  `json:"name"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	FPS      float64 `json:"fps"`
	Duration float64 `json:"duration"`
}

type TestTranscodeResult struct {
	Codec       string              `json:"codec"`
	Label       string              `json:"label"`
	Encoder     string              `json:"encoder"`
	Backend     string              `json:"backend"`
	Decoder     string              `json:"decoder,omitempty"` // GPU decoder; empty when the CPU decoded
	Sample      TestTranscodeSample `json:"sample"`
	Width       int                 `json:"width"`
	Height      int                 `json:"height"`
	FPS         float64             `json:"fps"`
	Speed       float64             `json:"speed"`
	OutputBytes int64               `json:"output_bytes"`
	DurationMS  int64               `json:"duration_ms"`
	Success     bool                `json:"success"`
	Warning     string              `json:"warning,omitempty"`
	Error       string              `json:"error,omitempty"`
	Notes       []string            `json:"notes,omitempty"`
}

type TestTranscodeResponse struct {
	Hardware transcode.Hardware    `json:"hardware"`
	Results  []TestTranscodeResult `json:"results"`
	Notes    []string              `json:"notes,omitempty"`
}

func (i ConfigResource) testTranscode(req *restful.Request, resp *restful.Response) {
	var r RequestTestTranscode
	if err := req.ReadEntity(&r); err != nil && err != io.EOF {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	codecs := transcode.ParseCodecs(r.Codecs)
	if len(codecs) == 0 {
		codecs = transcode.ParseCodecs(config.Config.Interfaces.Heresphere.TranscodeCodecs)
	}
	if len(codecs) == 0 {
		codecs = []string{transcode.CodecH264}
	}
	limit := r.SampleLimit
	if limit <= 0 {
		limit = 3
	}
	if limit > 5 {
		limit = 5
	}
	duration := r.DurationSecond
	if duration <= 0 {
		duration = 2
	}
	if duration < 0.5 {
		duration = 0.5
	}
	if duration > 10 {
		duration = 10
	}

	ffmpeg, err := transcode.EnsureManagedFFmpeg(common.BinDir)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusOK, TestTranscodeResponse{
			Notes: []string{fmt.Sprintf("Could not get jellyfin-ffmpeg %s: %v", transcode.JellyfinFFmpegVersion, err)},
		})
		return
	}
	hw := transcode.DetectFresh(ffmpeg)
	// Test at the highest offered bitrate, the hardest for the encoder to keep up with.
	bitrates := transcodeBitrates()
	opts := transcodeOptions(bitrates[len(bitrates)-1])

	db, err := models.GetDB()
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer db.Close()
	samples := pickTranscodeSamples(db, limit)

	out := TestTranscodeResponse{Hardware: hw}
	if len(samples) == 0 {
		out.Notes = append(out.Notes, "No local video files with resolution metadata were found to test")
		log.Warnf("transcode test: no samples available")
		resp.WriteHeaderAndEntity(http.StatusOK, out)
		return
	}

	for _, file := range samples {
		sample := TestTranscodeSample{
			FileID:   file.ID,
			Name:     file.Filename,
			Width:    file.VideoWidth,
			Height:   file.VideoHeight,
			FPS:      file.VideoAvgFrameRateVal,
			Duration: file.VideoDuration,
		}
		for _, codec := range codecs {
			plan := transcode.PlanForEncoder(codec, hw, opts, file.VideoWidth, file.VideoHeight, file.VideoAvgFrameRateVal)
			plan = withGPUDecode(plan, file)
			result := runTranscodeTest(ffmpeg, file, sample, codec, plan, opts, duration)
			if !result.Success && plan.Decoder != "" {
				// Streams retry on the CPU when the GPU rejects a source, so the test does too.
				failed := result.Error
				result = runTranscodeTest(ffmpeg, file, sample, codec, plan.CPUDecode(), opts, duration)
				note := "decoding with " + plan.Decoder + " failed, so the CPU decoded: " + failed
				if result.Warning != "" {
					note = result.Warning + "; " + note
				}
				result.Warning = note
			}
			out.Results = append(out.Results, result)
			if result.Success {
				log.Infof("transcode test %s %s %dx%d: ok at %.1fx for %.2fs", file.Filename, result.Label, result.Width, result.Height, result.Speed, duration)
				if result.Warning != "" {
					log.Warnf("transcode test %s %s: %s", file.Filename, result.Label, result.Warning)
				}
			} else {
				log.Warnf("transcode test %s %s %dx%d: %s", file.Filename, result.Label, result.Width, result.Height, result.Error)
			}
		}
	}

	resp.WriteHeaderAndEntity(http.StatusOK, out)
}

func pickTranscodeSamples(db *gorm.DB, limit int) []models.File {
	base := db.Model(&models.File{}).
		Joins("JOIN volumes ON volumes.id = files.volume_id").
		Where("files.type = ?", "video").
		Where("files.video_width > 0 AND files.video_height > 0 AND files.video_duration > 0").
		Where("volumes.type = ?", "local").
		Order("files.video_width * files.video_height ASC")

	var count int
	if err := base.Count(&count).Error; err != nil || count == 0 {
		return nil
	}

	// Take the smallest, median and largest pixel counts first. The quartiles fill in if
	// those rows share a resolution or point at a missing file.
	positions := []int{0, count / 2, count - 1, count / 4, count * 3 / 4}
	var out []models.File
	seen := map[string]bool{}
	for _, pos := range positions {
		if len(out) >= limit {
			break
		}
		if pos < 0 {
			pos = 0
		}
		if pos >= count {
			pos = count - 1
		}

		var file models.File
		err := db.Model(&models.File{}).
			Joins("JOIN volumes ON volumes.id = files.volume_id").
			Where("files.type = ?", "video").
			Where("files.video_width > 0 AND files.video_height > 0 AND files.video_duration > 0").
			Where("volumes.type = ?", "local").
			Preload("Volume").
			Order("files.video_width * files.video_height ASC").
			Offset(pos).
			Limit(1).
			Find(&file).Error
		if err != nil || file.ID == 0 {
			continue
		}
		key := fmt.Sprintf("%dx%d", file.VideoWidth, file.VideoHeight)
		if seen[key] {
			continue
		}
		if _, err := os.Stat(file.GetPath()); err != nil {
			continue
		}
		seen[key] = true
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].VideoWidth*out[i].VideoHeight < out[j].VideoWidth*out[j].VideoHeight
	})
	return out
}

func runTranscodeTest(ffmpeg string, file models.File, sample TestTranscodeSample, codec string, plan transcode.Plan, opts transcode.Options, duration float64) TestTranscodeResult {
	result := TestTranscodeResult{
		Codec:   codec,
		Label:   plan.Label(),
		Encoder: plan.Encoder,
		Backend: plan.Backend.Name,
		Decoder: plan.Decoder,
		Sample:  sample,
		Width:   plan.Width,
		Height:  plan.Height,
		Notes:   plan.Notes,
	}
	if result.Width <= 0 {
		result.Width = sample.Width
	}
	if result.Height <= 0 {
		result.Height = sample.Height
	}
	if plan.Unavailable {
		result.Error = strings.Join(plan.Notes, "; ")
		return result
	}
	if plan.Encoder == "" {
		result.Error = "encoder is not available"
		return result
	}

	dir, err := os.MkdirTemp("", "xbvr-transcode-test-")
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer os.RemoveAll(dir)
	outPath := filepath.Join(dir, "sample.mp4")

	args := transcode.TestArgs(ffmpeg, file.GetPath(), duration, opts, plan, outPath)
	ctxTimeout := time.Duration(duration*30+45) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	result.DurationMS = time.Since(start).Milliseconds()
	result.FPS, result.Speed = parseProgress(stdout.String())

	if stat, statErr := os.Stat(outPath); statErr == nil {
		result.OutputBytes = stat.Size()
	}
	if err != nil {
		result.Error = ffmpegError(stderr.String())
		return result
	}
	if result.OutputBytes == 0 {
		result.Error = "ffmpeg produced an empty file"
		return result
	}
	result.Success = true
	if result.Speed > 0 && result.Speed < 0.95 {
		result.Warning = fmt.Sprintf("only %.2fx realtime at this size and frame rate; playback may stutter", result.Speed)
	}
	return result
}

func parseProgress(out string) (fps, speed float64) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "fps":
			if v, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
				fps = v
			}
		case "speed":
			value := strings.TrimSuffix(strings.TrimSpace(fields[1]), "x")
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				speed = v
			}
		}
	}
	return fps, speed
}

func ffmpegError(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	var kept []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			kept = append(kept, line)
		}
	}
	if len(kept) > 4 {
		kept = kept[len(kept)-4:]
	}
	if len(kept) == 0 {
		return "ffmpeg failed"
	}
	return strings.Join(kept, "; ")
}
