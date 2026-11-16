package transcode

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BackendNVENC = "nvenc"
	BackendQSV   = "qsv"
	BackendVAAPI = "vaapi"
	BackendCPU   = "cpu"

	hardwareCacheTTL    = 2 * time.Minute
	hardwareProbeLimit  = 20 * time.Second
	hardwareProbeWidth  = 320
	hardwareProbeHeight = 240
)

// Backend describes one way of encoding video: the ffmpeg encoder names it can use,
// the device it needs, and the GPU (for NVENC) it should select. The same codec can
// exist on more than one backend, so the backend is chosen separately from the codec.
type Backend struct {
	Name        string            `json:"name"`
	Label       string            `json:"label"`
	Encoders    map[string]string `json:"encoders"`
	Device      string            `json:"device,omitempty"`
	GPUIndex    int               `json:"gpu_index,omitempty"`
	FreeVRAMMiB int               `json:"free_vram_mib,omitempty"`
	Notes       []string          `json:"notes,omitempty"`

	// Some Intel systems can only create a QSV device from an existing VAAPI device.
	qsvViaVAAPI bool
}

// Supports reports whether the backend has a working encoder for codec.
func (b Backend) Supports(codec string) bool {
	_, ok := b.Encoders[strings.ToLower(strings.TrimSpace(codec))]
	return ok
}

// Encoder returns the ffmpeg encoder for codec on this backend.
func (b Backend) Encoder(codec string) string {
	return b.Encoders[strings.ToLower(strings.TrimSpace(codec))]
}

// LabelFor names an encoder for players that show the stream's codec.
func (b Backend) LabelFor(codec string) string {
	switch b.Name {
	case BackendNVENC:
		switch codec {
		case CodecHEVC:
			return "HEVC (NVENC)"
		case CodecAV1:
			return "AV1 (NVENC)"
		default:
			return "H.264 (NVENC)"
		}
	case BackendQSV:
		switch codec {
		case CodecHEVC:
			return "HEVC (Intel Quick Sync)"
		case CodecAV1:
			return "AV1 (Intel Quick Sync)"
		default:
			return "H.264 (Intel Quick Sync)"
		}
	case BackendVAAPI:
		switch codec {
		case CodecHEVC:
			return "HEVC (VAAPI)"
		case CodecAV1:
			return "AV1 (VAAPI)"
		default:
			return "H.264 (VAAPI)"
		}
	case BackendCPU:
		return "H.264 (x264)"
	default:
		return CodecLabel(codec)
	}
}

// preInputArgs are the hardware device options that have to appear before the input.
func (b Backend) preInputArgs() []string {
	switch b.Name {
	case BackendVAAPI:
		if b.Device == "" {
			return nil
		}
		return []string{"-vaapi_device", b.Device}
	case BackendQSV:
		if b.qsvViaVAAPI {
			if b.Device == "" {
				return []string{"-init_hw_device", "qsv=hw", "-filter_hw_device", "hw"}
			}
			return []string{
				"-init_hw_device", "vaapi=va:" + b.Device,
				"-init_hw_device", "qsv=hw@va",
				"-filter_hw_device", "hw",
			}
		}
		if b.Device == "" {
			return []string{"-init_hw_device", "qsv=hw", "-filter_hw_device", "hw"}
		}
		return []string{"-init_hw_device", "qsv=hw:" + b.Device, "-filter_hw_device", "hw"}
	default:
		return nil
	}
}

// Hardware is the result of probing the encoders on this machine. Preferred is the
// backend used first; Available keeps every backend that passed the probe so a
// machine with more than one GPU can still offer a codec the preferred one lacks.
type Hardware struct {
	Preferred Backend   `json:"preferred"`
	Available []Backend `json:"available"`
	FFmpeg    string    `json:"ffmpeg,omitempty"`
	Notes     []string  `json:"notes,omitempty"`

	// InitialBurst reports that this ffmpeg can read ahead of realtime at the start of an
	// input. Older builds reject the option outright.
	InitialBurst bool `json:"-"`
}

// BackendFor picks the first usable backend for codec, preferring the overall best one.
func (h Hardware) BackendFor(codec string) (Backend, bool) {
	codec = strings.ToLower(strings.TrimSpace(codec))
	if h.Preferred.Supports(codec) {
		return h.Preferred, true
	}
	for _, b := range h.Available {
		if b.Supports(codec) {
			return b, true
		}
	}
	return Backend{}, false
}

// Codecs returns the codecs available through the preferred backend. It is used for
// diagnostics and for deciding whether the option should be offered at all.
func (h Hardware) Codecs() []string {
	var out []string
	for _, codec := range codecOrder {
		if h.Preferred.Supports(codec) {
			out = append(out, codec)
		}
	}
	return out
}

var hardwareCache struct {
	sync.Mutex
	key string
	at  time.Time
	hw  Hardware
}

// Detect returns the hardware encoders this machine can use. The result is cached
// briefly because a stream request may otherwise spend time probing before playback.
func Detect(ffmpegPath string) Hardware {
	ffmpegPath = resolveFFmpeg(ffmpegPath)

	hardwareCache.Lock()
	defer hardwareCache.Unlock()

	if hardwareCache.key == ffmpegPath && time.Since(hardwareCache.at) < hardwareCacheTTL {
		return hardwareCache.hw
	}

	hw := probeHardware(ffmpegPath)
	hardwareCache.key = ffmpegPath
	hardwareCache.at = time.Now()
	hardwareCache.hw = hw
	return hw
}

// DetectFresh probes again instead of using the cache, for the setup test button.
func DetectFresh(ffmpegPath string) Hardware {
	ffmpegPath = resolveFFmpeg(ffmpegPath)

	hardwareCache.Lock()
	defer hardwareCache.Unlock()

	hw := probeHardware(ffmpegPath)
	hardwareCache.key = ffmpegPath
	hardwareCache.at = time.Now()
	hardwareCache.hw = hw
	return hw
}

// RefreshVRAM updates the cached NVENC free-memory figure before a new stream starts.
// Encoder availability changes rarely, but the free memory can change while a stream or
// another CUDA workload is running.
func RefreshVRAM(hw Hardware) Hardware {
	if hw.Preferred.Name != BackendNVENC {
		return hw
	}
	gpus := nvidiaGPUs()
	if len(gpus) == 0 {
		return hw
	}
	update := func(index int) (int, bool) {
		for _, gpu := range gpus {
			if gpu.index == index {
				return gpu.free, true
			}
		}
		return 0, false
	}
	if free, ok := update(hw.Preferred.GPUIndex); ok {
		hw.Preferred.FreeVRAMMiB = free
	}
	for i := range hw.Available {
		if hw.Available[i].Name != BackendNVENC {
			continue
		}
		if free, ok := update(hw.Available[i].GPUIndex); ok {
			hw.Available[i].FreeVRAMMiB = free
		}
	}
	return hw
}

func resolveFFmpeg(path string) string {
	if strings.TrimSpace(path) != "" {
		return path
	}
	if found, err := exec.LookPath("ffmpeg"); err == nil {
		return found
	}
	return "ffmpeg"
}

// supportsInitialBurst reports whether ffmpeg accepts -readrate_initial_burst.
func supportsInitialBurst(ffmpegPath string) bool {
	out, err := exec.Command(ffmpegPath, "-hide_banner", "-h", "long").CombinedOutput()
	return err == nil && strings.Contains(string(out), "readrate_initial_burst")
}

func probeHardware(ffmpegPath string) Hardware {
	hw := Hardware{FFmpeg: ffmpegPath, InitialBurst: supportsInitialBurst(ffmpegPath)}
	encoders, err := listEncoders(ffmpegPath)
	if err != nil {
		hw.Notes = append(hw.Notes, "Could not list ffmpeg encoders: "+err.Error())
		return hw
	}

	if backend, notes := probeNVENC(ffmpegPath, encoders); backend.Name != "" {
		hw.Available = append(hw.Available, backend)
	} else if hasAnyEncoder(encoders, "h264_nvenc", "hevc_nvenc", "av1_nvenc") {
		hw.Notes = append(hw.Notes, notes...)
	}

	if backend, notes := probeQSV(ffmpegPath, encoders); backend.Name != "" {
		hw.Available = append(hw.Available, backend)
	} else if hasAnyEncoder(encoders, "h264_qsv", "hevc_qsv", "av1_qsv") {
		hw.Notes = append(hw.Notes, notes...)
	}

	if backend, notes := probeVAAPI(ffmpegPath, encoders); backend.Name != "" {
		hw.Available = append(hw.Available, backend)
	} else if hasAnyEncoder(encoders, "h264_vaapi", "hevc_vaapi", "av1_vaapi") {
		hw.Notes = append(hw.Notes, notes...)
	}

	if encoders["libx264"] {
		hw.Available = append(hw.Available, Backend{
			Name:     BackendCPU,
			Label:    "CPU (x264)",
			Encoders: map[string]string{CodecH264: "libx264", CodecX264: "libx264"},
			Notes:    []string{"CPU encode; high resolutions may not keep up with playback"},
		})
	}

	if len(hw.Available) == 0 {
		hw.Notes = append(hw.Notes, "No usable video encoder was found")
		return hw
	}

	// Prefer dedicated encoders over the CPU fallback. QSV is preferred over plain
	// VAAPI on Intel because it exposes the more complete encode path.
	rank := map[string]int{BackendNVENC: 1, BackendQSV: 2, BackendVAAPI: 3, BackendCPU: 4}
	sort.SliceStable(hw.Available, func(i, j int) bool {
		ri, rj := rank[hw.Available[i].Name], rank[hw.Available[j].Name]
		if ri != rj {
			return ri < rj
		}
		return hw.Available[i].FreeVRAMMiB > hw.Available[j].FreeVRAMMiB
	})
	hw.Preferred = hw.Available[0]
	return hw
}

func listEncoders(ffmpegPath string) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hardwareProbeLimit)
	defer cancel()

	out, err := exec.CommandContext(ctx, ffmpegPath, "-hide_banner", "-encoders").Output()
	if err != nil {
		return nil, err
	}

	encoders := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "V") {
			continue
		}
		encoders[fields[1]] = true
	}
	return encoders, nil
}

func hasAnyEncoder(encoders map[string]bool, names ...string) bool {
	for _, name := range names {
		if encoders[name] {
			return true
		}
	}
	return false
}

func nvencEncoders(encoders map[string]bool) map[string]string {
	names := map[string]string{CodecH264: "h264_nvenc", CodecHEVC: "hevc_nvenc", CodecAV1: "av1_nvenc"}
	out := map[string]string{}
	for codec, name := range names {
		if encoders[name] {
			out[codec] = name
		}
	}
	return out
}

func qsvEncoders(encoders map[string]bool) map[string]string {
	names := map[string]string{CodecH264: "h264_qsv", CodecHEVC: "hevc_qsv", CodecAV1: "av1_qsv"}
	out := map[string]string{}
	for codec, name := range names {
		if encoders[name] {
			out[codec] = name
		}
	}
	return out
}

func vaapiEncoders(encoders map[string]bool) map[string]string {
	names := map[string]string{CodecH264: "h264_vaapi", CodecHEVC: "hevc_vaapi", CodecAV1: "av1_vaapi"}
	out := map[string]string{}
	for codec, name := range names {
		if encoders[name] {
			out[codec] = name
		}
	}
	return out
}

type nvidiaGPU struct {
	index int
	free  int
}

func nvidiaGPUs() []nvidiaGPU {
	out, err := exec.Command("nvidia-smi", "--query-gpu=index,memory.free", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil
	}
	var gpus []nvidiaGPU
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) != 2 {
			continue
		}
		idx, err1 := strconv.Atoi(strings.TrimSpace(fields[0]))
		free, err2 := strconv.Atoi(strings.TrimSpace(fields[1]))
		if err1 != nil || err2 != nil {
			continue
		}
		gpus = append(gpus, nvidiaGPU{index: idx, free: free})
	}
	sort.SliceStable(gpus, func(i, j int) bool { return gpus[i].free > gpus[j].free })
	return gpus
}

func drmDevices() []string {
	render, _ := filepath.Glob("/dev/dri/renderD*")
	sort.Strings(render)
	if len(render) > 0 {
		return render
	}
	card, _ := filepath.Glob("/dev/dri/card*")
	sort.Strings(card)
	return card
}

func probeNVENC(ffmpegPath string, encoders map[string]bool) (Backend, []string) {
	names := nvencEncoders(encoders)
	if len(names) == 0 {
		return Backend{}, nil
	}

	gpus := nvidiaGPUs()
	if len(gpus) == 0 {
		// NVENC can still be present when nvidia-smi is not on PATH.
		gpus = []nvidiaGPU{{index: 0}}
	}

	var best Backend
	bestCount := -1
	var firstErr error
	for _, gpu := range gpus {
		candidate := Backend{
			Name:        BackendNVENC,
			Label:       "NVENC",
			Encoders:    map[string]string{},
			GPUIndex:    gpu.index,
			FreeVRAMMiB: gpu.free,
		}
		for codec, encoder := range names {
			if err := probeEncoder(ffmpegPath, candidate, codec, encoder); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			candidate.Encoders[codec] = encoder
		}
		if len(candidate.Encoders) > bestCount {
			best, bestCount = candidate, len(candidate.Encoders)
		}
	}
	if bestCount <= 0 {
		if firstErr != nil {
			return Backend{}, []string{"NVENC: " + compactError(firstErr)}
		}
		return Backend{}, nil
	}
	return best, nil
}

func probeQSV(ffmpegPath string, encoders map[string]bool) (Backend, []string) {
	names := qsvEncoders(encoders)
	if len(names) == 0 {
		return Backend{}, nil
	}

	devices := drmDevices()
	if len(devices) == 0 {
		return Backend{}, []string{"Quick Sync is compiled but /dev/dri has no render device"}
	}
	var firstErr error
	for _, device := range devices {
		for _, viaVAAPI := range []bool{false, true} {
			candidate := Backend{
				Name:        BackendQSV,
				Label:       "Intel Quick Sync",
				Encoders:    map[string]string{},
				Device:      device,
				FreeVRAMMiB: 0,
				qsvViaVAAPI: viaVAAPI,
			}
			var candidateErr error
			for codec, encoder := range names {
				if err := probeEncoder(ffmpegPath, candidate, codec, encoder); err != nil {
					if candidateErr == nil {
						candidateErr = err
					}
					continue
				}
				candidate.Encoders[codec] = encoder
			}
			if len(candidate.Encoders) > 0 {
				return candidate, nil
			}
			if candidateErr != nil && firstErr == nil {
				firstErr = candidateErr
			}
		}
	}
	if firstErr != nil {
		return Backend{}, []string{"Quick Sync: " + compactError(firstErr)}
	}
	return Backend{}, nil
}

func probeVAAPI(ffmpegPath string, encoders map[string]bool) (Backend, []string) {
	names := vaapiEncoders(encoders)
	if len(names) == 0 {
		return Backend{}, nil
	}

	devices := drmDevices()
	if len(devices) == 0 {
		return Backend{}, []string{"VAAPI is compiled but /dev/dri has no render device"}
	}
	var firstErr error
	for _, device := range devices {
		candidate := Backend{
			Name:     BackendVAAPI,
			Label:    "VAAPI",
			Encoders: map[string]string{},
			Device:   device,
		}
		var candidateErr error
		for codec, encoder := range names {
			if err := probeEncoder(ffmpegPath, candidate, codec, encoder); err != nil {
				if candidateErr == nil {
					candidateErr = err
				}
				continue
			}
			candidate.Encoders[codec] = encoder
		}
		if len(candidate.Encoders) > 0 {
			return candidate, nil
		}
		if candidateErr != nil && firstErr == nil {
			firstErr = candidateErr
		}
	}
	if firstErr != nil {
		return Backend{}, []string{"VAAPI: " + compactError(firstErr)}
	}
	return Backend{}, nil
}

func probeEncoder(ffmpegPath string, backend Backend, codec, encoder string) error {
	plan := Plan{
		Codec:   codec,
		Encoder: encoder,
		Backend: backend,
		Width:   hardwareProbeWidth,
		Height:  hardwareProbeHeight,
		FPS:     30,
	}
	args := []string{
		ffmpegPath, "-hide_banner", "-loglevel", "error", "-nostdin",
	}
	args = append(args, backend.preInputArgs()...)
	args = append(args,
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc2=size=%dx%d:rate=30:duration=0.3", hardwareProbeWidth, hardwareProbeHeight),
	)
	args = append(args, videoArgs(Options{BitrateMbps: 5, AudioKbps: 128}, plan)...)
	args = append(args, "-frames:v", "2", "-an", "-f", "null", "-")

	ctx, cancel := context.WithTimeout(context.Background(), hardwareProbeLimit)
	defer cancel()
	var stderr strings.Builder
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, compactError(fmt.Errorf("%s", stderr.String())))
	}
	return nil
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	line := strings.TrimSpace(err.Error())
	if line == "" {
		return "unknown error"
	}
	parts := strings.Split(line, "\n")
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	return strings.Join(parts, "; ")
}
