package organize

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

// DuplicateDismissal marks a scene's duplicates as ignored (legacy, per-scene).
type DuplicateDismissal struct {
	SceneID uint `gorm:"primary_key" json:"sceneId"`
}

// DuplicateFileIgnore hides a single file from the duplicate list.
type DuplicateFileIgnore struct {
	FileID uint `gorm:"primary_key" json:"fileId"`
}

// DuplicateReport caches the analysis for one scene (Report is a JSON DupGroup).
type DuplicateReport struct {
	SceneID    uint      `gorm:"primary_key" json:"sceneId"`
	Report     string    `gorm:"type:text" json:"-"`
	AnalyzedAt time.Time `json:"analyzedAt"`
}

// DupFile describes one file in a duplicate group.
type DupFile struct {
	FileID     uint    `json:"fileId"`
	Path       string  `json:"path"`
	Filename   string  `json:"filename"`
	Height     int     `json:"height"`
	Width      int     `json:"width"`
	Bitrate    int     `json:"bitrate"`
	Size       int64   `json:"size"`
	Duration   float64 `json:"duration"`
	Projection string  `json:"projection"`
	PSNR       float64 `json:"psnr"`    // vs the reference file (0 for the reference itself)
	Suggest    string  `json:"suggest"` // "keep" | "delete" | "review"
	Ignored    bool    `json:"ignored"`
}

// DupGroup is the analysis result for one scene with multiple distinct video files.
type DupGroup struct {
	SceneID          uint      `json:"sceneId"`
	Title            string    `json:"title"`
	Site             string    `json:"site"`
	Files            []DupFile `json:"files"`
	KeepFileID       uint      `json:"keepFileId"`
	DurationSpread   float64   `json:"durationSpread"`
	DurationMismatch bool      `json:"durationMismatch"`
	Status           string    `json:"status"`
	Detail           string    `json:"detail"`
	AnalyzedAt       time.Time `json:"analyzedAt"`
}

const psnrSampleSeconds = 2

var (
	dupMu      sync.Mutex
	dupRunning bool
	dupDone    int
	dupTotal   int
	psnrLineRe = regexp.MustCompile(`average:([0-9.]+|inf|nan)`)
)

func ffmpegBin() string {
	p := filepath.Join(common.BinDir, "ffmpeg")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return "ffmpeg"
}

// StartDupAnalysis runs the (expensive) duplicate analysis in the background.
func StartDupAnalysis(force bool) bool {
	// Shared with the organize runner so the two can't touch the same files at once.
	if !busy.TryLock() {
		return false
	}
	dupMu.Lock()
	dupRunning = true
	dupDone, dupTotal = 0, 0
	dupMu.Unlock()

	go func() {
		defer busy.Unlock()
		db, err := models.GetDB()
		if err == nil {
			analyzeDuplicates(db, force)
			db.Close()
		}
		dupMu.Lock()
		dupRunning = false
		dupMu.Unlock()
	}()
	return true
}

// DupStatus reports progress.
func DupStatus() (running bool, done, total int) {
	dupMu.Lock()
	defer dupMu.Unlock()
	return dupRunning, dupDone, dupTotal
}

func analyzeDuplicates(db *gorm.DB, force bool) {
	dismissed := map[uint]bool{}
	var dis []DuplicateDismissal
	db.Find(&dis)
	for _, d := range dis {
		dismissed[d.SceneID] = true
	}

	var ids []uint
	db.Model(&models.File{}).Where("type = ? AND scene_id > 0", "video").
		Group("scene_id").Having("count(*) > 1").Pluck("scene_id", &ids)

	dupMu.Lock()
	dupTotal = len(ids)
	dupMu.Unlock()

	for _, sid := range ids {
		dupMu.Lock()
		dupDone++
		dupMu.Unlock()
		if dismissed[sid] {
			continue
		}
		if !force {
			var existing DuplicateReport
			if db.Where("scene_id = ?", sid).First(&existing).Error == nil {
				continue // already analysed
			}
		}
		var scene models.Scene
		if db.Preload("Files").First(&scene, sid).Error != nil {
			continue
		}
		g := analyzeScene(&scene)
		if g == nil {
			// no longer a real duplicate group; drop any stale report
			db.Where("scene_id = ?", sid).Delete(&DuplicateReport{})
			continue
		}
		data, _ := json.Marshal(g)
		db.Save(&DuplicateReport{SceneID: sid, Report: string(data), AnalyzedAt: g.AnalyzedAt})
	}
	common.Log.Infof("organize: duplicate analysis complete (%d scenes examined)", len(ids))
}

// analyzeScene builds a DupGroup for a scene, or nil if it has < 2 distinct files.
func analyzeScene(scene *models.Scene) *DupGroup {
	// distinct video files on disk, keyed by (size, os_hash) to skip exact dups,
	// and by inode to skip hard links.
	seenKey := map[string]bool{}
	seenIno := map[[2]uint64]bool{}
	var files []DupFile
	for i := range scene.Files {
		f := &scene.Files[i]
		if f.Type != "video" {
			continue
		}
		fi, err := os.Stat(f.GetPath())
		if err != nil {
			continue
		}
		ino := inodeKey(fi)
		if seenIno[ino] {
			continue
		}
		key := strconv.FormatInt(fi.Size(), 10) + ":" + f.OsHash
		if f.OsHash != "" && seenKey[key] {
			continue
		}
		seenIno[ino] = true
		seenKey[key] = true
		files = append(files, DupFile{
			FileID: f.ID, Path: f.Path, Filename: f.Filename,
			Height: f.VideoHeight, Width: f.VideoWidth, Bitrate: f.VideoBitRate,
			Size: fi.Size(), Duration: f.VideoDuration, Projection: f.VideoProjection,
		})
	}
	if len(files) < 2 {
		return nil
	}

	// reference = best (height, bitrate, size)
	ref := 0
	for i := range files {
		a, b := files[i], files[ref]
		if a.Height > b.Height || (a.Height == b.Height && (a.Bitrate > b.Bitrate || (a.Bitrate == b.Bitrate && a.Size > b.Size))) {
			ref = i
		}
	}

	// duration spread
	minD, maxD := files[0].Duration, files[0].Duration
	for _, f := range files {
		if f.Duration < minD {
			minD = f.Duration
		}
		if f.Duration > maxD {
			maxD = f.Duration
		}
	}
	spread := maxD - minD
	mismatch := maxD > 0 && spread > maxFloat(2.0, 0.03*maxD)

	g := &DupGroup{
		SceneID: scene.ID, Title: scene.Title, Site: scene.Site,
		KeepFileID: files[ref].FileID, DurationSpread: spread,
		DurationMismatch: mismatch, AnalyzedAt: time.Now(),
	}

	refPath := filepath.Join(files[ref].Path, files[ref].Filename)
	worst := 999.0
	for i := range files {
		if i == ref {
			files[i].PSNR = 0
			files[i].Suggest = "keep"
			continue
		}
		if mismatch {
			files[i].Suggest = "review"
			continue
		}
		p := filepath.Join(files[i].Path, files[i].Filename)
		psnr := psnrBetween(refPath, p, minD)
		files[i].PSNR = round2(psnr)
		if psnr < worst {
			worst = psnr
		}
		if psnr < 20 {
			files[i].Suggest = "review"
		} else {
			files[i].Suggest = "delete"
		}
	}
	g.Files = files

	switch {
	case mismatch:
		g.Status = "duration-mismatch"
		g.Detail = "Files differ in length (" + dur(minD) + " vs " + dur(maxD) + ") — one may be mis-assigned to this scene. Review before deleting."
	case worst < 20:
		g.Status = "low-psnr"
		g.Detail = "Low PSNR — the files may be different content despite sharing a scene. Review."
	case worst >= 40:
		g.Status = "identical"
		g.Detail = "Near-identical content; keep the highest-quality file, the rest are safe to delete."
	default:
		g.Status = "same-content"
		g.Detail = "Same content at differing quality; keep the highest resolution/bitrate."
	}
	return g
}

// psnrBetween returns the average PSNR of b against reference a, sampled at a few
// aligned points and scaled to a common size. Returns 99 for identical (inf).
func psnrBetween(a, b string, minDur float64) float64 {
	if minDur <= 0 {
		minDur = 30
	}
	fracs := []float64{0.2, 0.5, 0.8}
	var vals []float64
	for _, fr := range fracs {
		t := int(minDur * fr)
		if v, ok := psnrSample(a, b, t); ok {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func psnrSample(a, b string, t int) (float64, bool) {
	const scale = "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=5"
	cmd := exec.Command(ffmpegBin(), "-hide_banner",
		"-ss", strconv.Itoa(t), "-t", strconv.Itoa(psnrSampleSeconds), "-i", a,
		"-ss", strconv.Itoa(t), "-t", strconv.Itoa(psnrSampleSeconds), "-i", b,
		"-lavfi", "[0:v]"+scale+"[r];[1:v]"+scale+"[d];[d][r]psnr", "-f", "null", "-")
	out, _ := cmd.CombinedOutput()
	m := psnrLineRe.FindSubmatch(out)
	if m == nil {
		return 0, false
	}
	s := string(m[1])
	if s == "inf" {
		return 99, true
	}
	if s == "nan" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// ListDupGroups returns the cached duplicate groups. Ignored files are hidden unless
// showIgnored is set; a group with fewer than two visible files is omitted.
func ListDupGroups(db *gorm.DB, showIgnored bool) []DupGroup {
	ignored := map[uint]bool{}
	var igs []DuplicateFileIgnore
	db.Find(&igs)
	for _, x := range igs {
		ignored[x.FileID] = true
	}
	var reps []DuplicateReport
	db.Order("analyzed_at desc").Find(&reps)
	out := []DupGroup{}
	for _, r := range reps {
		var g DupGroup
		if json.Unmarshal([]byte(r.Report), &g) != nil {
			continue
		}
		var files []DupFile
		nonIgnored := 0
		for _, f := range g.Files {
			f.Ignored = ignored[f.FileID]
			if f.Ignored && !showIgnored {
				continue
			}
			if !f.Ignored {
				nonIgnored++
			}
			files = append(files, f)
		}
		g.Files = files
		if !showIgnored && nonIgnored < 2 {
			continue // collapsed by ignores
		}
		if len(files) == 0 {
			continue
		}
		out = append(out, g)
	}
	return out
}

// IgnoreFile / UnignoreFile hide or restore a single file in the duplicate list.
func IgnoreFile(db *gorm.DB, fileID uint) {
	db.Save(&DuplicateFileIgnore{FileID: fileID})
}
func UnignoreFile(db *gorm.DB, fileID uint) {
	db.Delete(&DuplicateFileIgnore{}, fileID)
}

// DeleteFiles removes the given files from disk and DB, refreshing scene status.
func DeleteFiles(db *gorm.DB, ids []uint) int {
	n := 0
	scenes := map[uint]bool{}
	for _, id := range ids {
		var f models.File
		if db.First(&f, id).Error != nil {
			continue
		}
		if err := os.Remove(f.GetPath()); err != nil && !os.IsNotExist(err) {
			common.Log.Warnf("organize: delete %s failed: %v", f.GetPath(), err)
			continue
		}
		if f.SceneID != 0 {
			scenes[f.SceneID] = true
		}
		db.Delete(&models.File{}, id)
		n++
	}
	for sid := range scenes {
		var s models.Scene
		if db.First(&s, sid).Error == nil {
			s.UpdateStatus()
		}
		db.Where("scene_id = ?", sid).Delete(&DuplicateReport{})
	}
	common.Log.Infof("organize: deleted %d duplicate file(s)", n)
	return n
}

// DisassociateFiles detaches files from their scene (xbvr "unmatch"): scene_id -> 0,
// removes the filename from the scene's FilenamesArr so it won't be auto-rematched,
// records the action and refreshes scene status. The file stays on disk.
func DisassociateFiles(db *gorm.DB, ids []uint) int {
	n := 0
	scenes := map[uint]bool{}
	for _, id := range ids {
		var f models.File
		if db.Where(&models.File{ID: id}).First(&f).Error != nil {
			continue
		}
		sceneID := f.SceneID
		if sceneID == 0 {
			continue
		}
		f.SceneID = 0
		f.Save()

		var scene models.Scene
		if scene.GetIfExistByPK(sceneID) == nil {
			var arr []string
			if json.Unmarshal([]byte(scene.FilenamesArr), &arr) == nil {
				na := []string{}
				for _, fn := range arr {
					if fn != f.Filename {
						na = append(na, fn)
					}
				}
				if b, err := json.Marshal(na); err == nil {
					scene.FilenamesArr = string(b)
				}
			}
			models.AddAction(scene.SceneID, "unmatch", "filenames_arr", scene.FilenamesArr)
			scene.UpdateStatus()
			scenes[sceneID] = true
		}
		n++
	}
	for sid := range scenes {
		db.Where("scene_id = ?", sid).Delete(&DuplicateReport{})
	}
	common.Log.Infof("organize: disassociated %d file(s) from their scenes", n)
	return n
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
func dur(sec float64) string {
	m := int(sec) / 60
	s := int(sec) % 60
	return strconv.Itoa(m) + "m" + strconv.Itoa(s) + "s"
}
