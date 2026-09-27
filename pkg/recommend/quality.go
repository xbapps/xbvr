package recommend

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// No-reference visual quality: sample one frame every 5 minutes (from 5 min),
// run a Laplacian (edge) convolution at native resolution and read the mean edge
// energy (signalstats.YAVG). Higher = crisper/more detail. The median across the
// sampled frames is stored per file. Works with ffmpeg 4.x (no blurdetect needed).

const (
	vqSampleIntervalSec = 300 // one frame every 5 minutes
	vqFirstSampleSec    = 300 // first sample at 5 minutes
	vqConcurrency       = 2   // files measured in parallel (each ffmpeg is itself threaded)
)

var vqYavgRe = regexp.MustCompile(`lavfi\.signalstats\.YAVG=([0-9.]+)`)

func ffmpegBin() string {
	p := filepath.Join(common.BinDir, "ffmpeg")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return "ffmpeg"
}

func sampleTimestamps(durationSec float64, maxSamples int) []int {
	if durationSec <= 0 || maxSamples <= 0 {
		return nil
	}
	if durationSec < vqFirstSampleSec+1 {
		return []int{int(durationSec / 2)} // short file: one mid-point sample
	}
	var ts []int
	for t := vqFirstSampleSec; float64(t) < durationSec-2 && len(ts) < maxSamples; t += vqSampleIntervalSec {
		ts = append(ts, t)
	}
	return ts
}

// measureFrameEdgeEnergy returns the mean Laplacian edge energy of one frame at
// time t (native resolution).
func measureFrameEdgeEnergy(path string, t int) (float64, bool) {
	tmp, err := os.CreateTemp("", "xbvr-vq-*.txt")
	if err != nil {
		return 0, false
	}
	meta := tmp.Name()
	tmp.Close()
	defer os.Remove(meta)

	cmd := exec.Command(ffmpegBin(),
		"-hide_banner", "-loglevel", "error",
		"-ss", strconv.Itoa(t), "-i", path, "-frames:v", "1",
		"-vf", "format=gray,convolution=0 -1 0 -1 4 -1 0 -1 0,signalstats,metadata=print:file="+meta,
		"-f", "null", "-")
	if err := cmd.Run(); err != nil {
		return 0, false
	}
	data, err := os.ReadFile(meta)
	if err != nil {
		return 0, false
	}
	m := vqYavgRe.FindSubmatch(data)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// computeFileQuality samples a file and returns (median edge energy, sample count).
func computeFileQuality(f *models.File, maxSamples int) (float64, int) {
	path := f.GetPath()
	if _, err := os.Stat(path); err != nil {
		return 0, 0
	}
	var vals []float64
	for _, t := range sampleTimestamps(f.VideoDuration, maxSamples) {
		if v, ok := measureFrameEdgeEnergy(path, t); ok {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return 0, 0
	}
	sort.Float64s(vals)
	return median(vals), len(vals)
}

// vqJob / vqResult carry a file (and its scene) through the measurement workers.
type vqJob struct {
	sceneID uint
	file    *models.File
}
type vqResult struct {
	sceneID uint
	fileID  uint
	quality float64
	samples int
}

// applyVisualQuality measures the selected scenes' video files and re-ranks the two
// lists as each result lands: it persists every file's quality, keeps a running
// per-scene best, and after each new measurement re-normalizes and rewrites the
// affected scene scores in the DB. The base scores are already persisted, so the
// lists are live throughout and simply refine in place.
func applyVisualQuality(db *gorm.DB, scenes []models.Scene, topWatch, topDelete map[uint]float64, cfg recConfig) {
	sceneIndex := make(map[uint]*models.Scene, len(scenes))
	for i := range scenes {
		sceneIndex[scenes[i].ID] = &scenes[i]
	}

	// Running per-scene best quality. Seed with any already-cached file qualities.
	sceneBestQ := make(map[uint]float64)
	var jobs []vqJob
	collect := func(id uint) {
		s := sceneIndex[id]
		if s == nil {
			return
		}
		for fi := range s.Files {
			f := &s.Files[fi]
			if f.Type != "video" {
				continue
			}
			if f.VisualQualitySamples > 0 { // already measured (cached)
				if f.VisualQuality > sceneBestQ[id] {
					sceneBestQ[id] = f.VisualQuality
				}
				continue
			}
			jobs = append(jobs, vqJob{sceneID: id, file: f})
		}
	}
	for id := range topWatch {
		collect(id)
	}
	for id := range topDelete {
		collect(id)
	}

	// If some scenes already have cached quality, re-rank once up front.
	if len(sceneBestQ) > 0 {
		rerankByQuality(db, sceneBestQ, topWatch, topDelete, cfg)
	}
	if len(jobs) == 0 {
		return
	}

	in := make(chan vqJob)
	out := make(chan vqResult)
	var wg sync.WaitGroup
	for w := 0; w < vqConcurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range in {
				q, n := computeFileQuality(j.file, cfg.VQMaxSamples)
				out <- vqResult{sceneID: j.sceneID, fileID: j.file.ID, quality: q, samples: n}
			}
		}()
	}
	go func() {
		for _, j := range jobs {
			in <- j
		}
		close(in)
		wg.Wait()
		close(out)
	}()

	measured := 0
	for r := range out {
		now := time.Now()
		db.Model(&models.File{}).Where("id = ?", r.fileID).Updates(map[string]interface{}{
			"visual_quality":             r.quality,
			"visual_quality_samples":     r.samples,
			"visual_quality_computed_at": now,
		})
		measured++
		if r.quality > 0 && r.quality > sceneBestQ[r.sceneID] {
			sceneBestQ[r.sceneID] = r.quality
			rerankByQuality(db, sceneBestQ, topWatch, topDelete, cfg)
		}
	}
	common.Log.Infof("recommend: measured visual quality for %d files, re-ranked %d scenes",
		measured, len(sceneBestQ))
}

// rerankByQuality re-normalizes the measured scenes' quality to a [0,1] percentile and
// rewrites their scores in the DB (crisp -> up in watch, soft -> up in delete). Scores
// are always derived from the persisted base values, so repeated calls don't compound.
func rerankByQuality(db *gorm.DB, sceneBestQ map[uint]float64, topWatch, topDelete map[uint]float64, cfg recConfig) {
	relQ := percentileRanks(sceneBestQ)
	for id, r := range relQ {
		if base, ok := topWatch[id]; ok {
			db.Model(&models.Scene{}).Where("id = ?", id).
				Update("rec_watch_score", base*(1+cfg.WVisualQuality*(r-0.5)))
		} else if base, ok := topDelete[id]; ok {
			db.Model(&models.Scene{}).Where("id = ?", id).
				Update("rec_delete_score", base*(1+cfg.WVisualQuality*(0.5-r)))
		}
	}
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// percentileRanks maps each id to its rank in [0,1] (lowest value -> 0, highest -> 1).
func percentileRanks(m map[uint]float64) map[uint]float64 {
	out := make(map[uint]float64, len(m))
	n := len(m)
	if n == 0 {
		return out
	}
	if n == 1 {
		for id := range m {
			out[id] = 0.5
		}
		return out
	}
	type kv struct {
		id uint
		v  float64
	}
	arr := make([]kv, 0, n)
	for id, v := range m {
		arr = append(arr, kv{id, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v < arr[j].v })
	for i, e := range arr {
		out[e.id] = float64(i) / float64(n-1)
	}
	return out
}
