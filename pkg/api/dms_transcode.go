package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
	"github.com/xbapps/xbvr/pkg/transcode"
)

// getTranscode streams a live ffmpeg encode as though it were a seekable file. The stream has
// a fixed byte layout with a seek index, so a byte range names either a position in the index,
// which starts a new encode at that time, or the continuation of the running encode.
func (i DMSResource) getTranscode(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.Heresphere.TranscodeEnabled {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	id, err := strconv.Atoi(req.PathParameter("file-id"))
	if err != nil || id <= 0 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	enabled := transcode.ParseCodecs(config.Config.Interfaces.Heresphere.TranscodeCodecs)
	if len(enabled) == 0 {
		enabled = []string{transcode.CodecH264}
	}
	codec := strings.ToLower(strings.TrimSpace(req.QueryParameter("codec")))
	if codec == "" {
		codec = enabled[0]
	}
	allowed := false
	for _, c := range enabled {
		if c == codec {
			allowed = true
			break
		}
	}
	if !allowed {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	bitrate, ok := transcodeBitrate(req.QueryParameter("bitrate"))
	if !ok {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	if !validTranscodeToken(uint(id), codec, req.QueryParameter("token")) {
		resp.WriteHeader(http.StatusForbidden)
		return
	}

	db, _ := models.GetDB()
	defer db.Close()
	var f models.File
	if err := db.Preload("Volume").First(&f, id).Error; err != nil || f.Volume.Type != "local" {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	if _, err := os.Stat(f.GetPath()); err != nil {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	doNotTrack := req.QueryParameter("dnt")
	setDeoPlayerHost(req)
	session.TrackSessionFromFile(f, doNotTrack)

	if f.VideoDuration <= 0 {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	opts := transcodeOptions(bitrate)
	layout, err := transcode.NewLayout(f.VideoDuration, opts)
	if err != nil {
		log.Warnf("transcode %d: %v", f.ID, err)
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	rangeHeader := req.Request.Header.Get("Range")
	start, end, hasRange := parseByteRange(rangeHeader, layout.Total)
	if !hasRange && rangeBeyondEnd(rangeHeader, layout.Total) {
		resp.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", layout.Total))
		resp.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	offset := int64(0)
	if hasRange {
		offset = start
	}

	// The layout is part of the key: a rescan that changes the duration changes the layout,
	// and a stream's cached header only fits the layout it was made for.
	source := transcode.Source{
		Key: fmt.Sprintf("%s/%d/%s/%d/%d/%d/%d", transcodeClient(req.Request), f.ID, codec,
			opts.BitrateMbps, opts.AudioKbps, opts.MaxHeight, layout.Total),
		Duration: f.VideoDuration,
		Layout:   layout,
		Prepare:  func() (transcode.Spawner, error) { return prepareTranscode(f, codec, opts) },
	}
	playback, err := transcodeStreams.Open(req.Request.Context(), source, offset)
	if err == nil {
		defer playback.Close()
	}
	if errors.Is(err, transcode.ErrUnavailable) {
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if errors.Is(err, transcode.ErrBusy) {
		log.Warnf("transcode %d %s: the most streams allowed at once are already running; refusing", f.ID, codec)
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if errors.Is(err, transcode.ErrCannotResume) {
		log.Warnf("transcode %d %s: range %q cannot be resumed", f.ID, codec, rangeHeader)
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if req.Request.Context().Err() != nil {
		return
	}
	if err != nil {
		log.Errorf("transcode %d %s: %v", f.ID, codec, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Infof("transcode %d %s: %s range=%q chunk=%d seek=%.3f ua=%q",
		f.ID, codec, playback.Kind, rangeHeader, playback.Chunk, playback.Seek, req.Request.UserAgent())

	var limit int64
	h := resp.Header()
	h.Set("Content-Type", "video/mp4")
	h.Set("Accept-Ranges", "bytes")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Duration", strconv.FormatFloat(f.VideoDuration, 'f', 3, 64))
	if hasRange {
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, layout.Total))
		if end < layout.Total-1 {
			limit = end - start + 1
			h.Set("Content-Length", strconv.FormatInt(limit, 10))
		}
		resp.WriteHeader(http.StatusPartialContent)
	} else {
		resp.WriteHeader(http.StatusOK)
	}

	// Headers go out before the encoder has produced anything, so the client's connection
	// timeout does not include encoder startup.
	writer := &flushWriter{w: resp.ResponseWriter}
	if flusher, ok := resp.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
	dest := io.Writer(writer)
	if limit > 0 {
		dest = &limitedWriter{w: writer, remaining: limit}
	}

	began := time.Now()
	sent, streamErr := playback.WriteTo(req.Request.Context(), dest)
	if errors.Is(streamErr, io.ErrShortWrite) {
		streamErr = nil
	}
	log.Infof("transcode %d %s: %s sent %d bytes in %s (%v)",
		f.ID, codec, playback.Kind, sent, time.Since(began).Round(time.Millisecond), streamErr)
	if streamFailed(req.Request.Context(), streamErr) {
		// Returning normally would end the chunked body cleanly, which a player reads as the
		// end of the video. Breaking the connection makes it retry instead.
		panic(http.ErrAbortHandler)
	}
}

// streamFailed reports whether a response ended early for a reason the client did not cause.
func streamFailed(ctx context.Context, err error) bool {
	return err != nil && ctx.Err() == nil
}

// transcodeClient identifies a viewer, so two viewers of the same file get separate encodes.
func transcodeClient(req *http.Request) string {
	if ip := strings.TrimSpace(req.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if forwarded := req.Header.Get("X-Forwarded-For"); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

// transcodeStreams holds the running encodes for every file and codec.
var transcodeStreams = newTranscodeManager()

func newTranscodeManager() *transcode.Manager {
	m := transcode.NewManager(func(format string, args ...interface{}) {
		log.Warnf(format, args...)
	})
	m.MaxEncodes = func() int { return config.Config.Interfaces.Heresphere.TranscodeMaxStreams }
	return m
}

// transcodeToken ties a transcode URL to the player login. While the login is on, only a
// player that fetched the scene with it has a URL that can start encodes.
func transcodeToken(fileID uint, codec string) string {
	mac := hmac.New(sha256.New, []byte("xbvr transcode\x00"+config.Config.Interfaces.DeoVR.Password))
	fmt.Fprintf(mac, "%d/%s", fileID, codec)
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

func validTranscodeToken(fileID uint, codec, token string) bool {
	if !isDeoAuthEnabled() {
		return true
	}
	return hmac.Equal([]byte(token), []byte(transcodeToken(fileID, codec)))
}

// prepareTranscode chooses the encoder for a stream. Its plan, including the output size, then
// stays fixed for every encode of that stream, so seeks cannot change the picture format.
func prepareTranscode(f models.File, codec string, opts transcode.Options) (transcode.Spawner, error) {
	hw := transcode.RefreshVRAM(transcode.Detect(ffmpegBin()))
	plan := transcode.PlanForEncoder(codec, hw, opts, f.VideoWidth, f.VideoHeight, f.VideoAvgFrameRateVal)
	if len(plan.Notes) > 0 {
		log.Infof("transcode %d %s on %s: %s", f.ID, codec, plan.Label(), strings.Join(plan.Notes, "; "))
	}
	if plan.Unavailable {
		log.Warnf("transcode %d %s: %s; refusing", f.ID, codec, strings.Join(plan.Notes, "; "))
		return nil, transcode.ErrUnavailable
	}
	path := f.GetPath()
	return func(ctx context.Context, seekSeconds float64) (*transcode.Process, error) {
		// Encodes start one at a time, so the next stream planned sees this one's memory as
		// used until ffmpeg has allocated it.
		release := plan.ReserveVRAM()
		args := transcode.Args(ffmpegBin(), path, seekSeconds, opts, plan, -1)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			release()
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			release()
			return nil, fmt.Errorf("starting ffmpeg: %w", err)
		}
		return &transcode.Process{
			Stdout: &releaseOnOutput{r: stdout, release: release},
			Wait: func() error {
				defer release()
				if err := cmd.Wait(); err != nil {
					return fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(stderr.String()))
				}
				return nil
			},
		}, nil
	}, nil
}

// releaseOnOutput calls release once ffmpeg writes anything. ffmpeg writes its header only after
// opening the encoders, so by then the encode's GPU memory shows in nvidia-smi.
type releaseOnOutput struct {
	r       io.Reader
	release func()
}

func (o *releaseOnOutput) Read(p []byte) (int, error) {
	n, err := o.r.Read(p)
	if n > 0 {
		o.release()
	}
	return n, err
}

// limitedWriter stops a stream at a byte count without buffering the rest of the encode.
type limitedWriter struct {
	w         io.Writer
	remaining int64
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		return 0, io.ErrShortWrite
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.w.Write(p)
	l.remaining -= int64(n)
	if err == nil && l.remaining == 0 {
		err = io.ErrShortWrite
	}
	return n, err
}

// ffmpegBin prefers an ffmpeg on PATH, so an install can provide a build with the hardware
// encoders it needs; it falls back to the bundled download when there is none.
func ffmpegBin() string {
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}
	return tasks.GetBinPath("ffmpeg")
}

// flushWriter pushes each chunk to the client so playback starts without a server buffer delay.
type flushWriter struct {
	w http.ResponseWriter
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fl, ok := fw.w.(http.Flusher); ok {
		fl.Flush()
	}
	return n, err
}

// parseByteRange parses one HTTP byte range against total. ok is false when there is no usable
// range header, in which case the caller serves the whole stream.
func parseByteRange(header string, total int64) (start, end int64, ok bool) {
	header = strings.TrimSpace(header)
	if header == "" || total <= 0 || !strings.HasPrefix(header, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimPrefix(header, "bytes=")
	if i := strings.Index(spec, ","); i >= 0 {
		spec = spec[:i]
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	if parts[0] == "" {
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false
		}
		start = total - suffix
		if start < 0 {
			start = 0
		}
		return start, total - 1, true
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= total {
		return 0, 0, false
	}
	if parts[1] == "" {
		return start, total - 1, true
	}
	end, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end < start {
		return 0, 0, false
	}
	if end >= total {
		end = total - 1
	}
	return start, end, true
}

// rangeBeyondEnd reports whether a byte range header asks for a position past the stream.
func rangeBeyondEnd(header string, total int64) bool {
	spec, found := strings.CutPrefix(strings.TrimSpace(header), "bytes=")
	if !found {
		return false
	}
	first, _, _ := strings.Cut(spec, "-")
	start, err := strconv.ParseInt(strings.TrimSpace(first), 10, 64)
	return err == nil && start >= total
}
