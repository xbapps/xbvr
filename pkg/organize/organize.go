// Package organize reorganises xbvr-managed VR videos into a canonical folder layout
// and keeps the files table in sync. It is a Go port of the standalone organize_vr.py
// tool, running inside xbvr so it can update the DB (and paths are already the
// container's volume paths, so no host mapping is needed).
//
// Target layout: ASeries/{Studio}/{Studio}.{YY}.{MM}.{DD}.{Cast}.{Title}.XXX.{FOV}.{Height}p/
package organize

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/xbapps/xbvr/pkg/models"
)

const maxCast = 4

var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".wmv": true, ".avi": true,
	".mov": true, ".m4v": true, ".ts": true, ".webm": true,
}

var sidecarExts = map[string]bool{
	".srt": true, ".ass": true, ".ssa": true, ".vtt": true, ".sub": true, ".smi": true, ".idx": true,
	".funscript": true, ".hsp": true,
	".nfo": true, ".json": true, ".txt": true,
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".bmp": true,
}

var fovMap = map[string]string{
	"180_sbs": "VR180", "360_tb": "VR360", "mkx200": "MKX200", "mkx220": "MKX220",
	"vrca220": "VRCA220", "fisheye190": "FISHEYE190", "fisheye": "FISHEYE", "rf52": "RF52",
}

// Options controls a run.
type Options struct {
	DryRun         bool   `json:"dryRun"`
	Limit          int    `json:"limit"`          // 0 = all; else consider at most N video files
	Dedup          bool   `json:"dedup"`          // delete byte-identical copies
	DeferDups      bool   `json:"deferDups"`      // skip scenes with possible dups
	IncomingDir    string `json:"incomingDir"`    // staging dir; "" disables
	IncomingMinAge int    `json:"incomingMinAge"` // days
	TopFolder      string `json:"topFolder"`      // wrapper folder under the storage-folder root; "" = studio folders directly in the root
	CastGender     string `json:"castGender"`     // folder-name cast preference: "any", "female", or "male"
	SymlinkByActor bool   `json:"symlinkByActor"` // also symlink each scene dir into per-actor folders
	ActorFolder    string `json:"actorFolder"`    // parent folder for the per-actor symlink dirs
}

// Action is one planned/performed operation (for the UI preview).
type Action struct {
	SceneID uint   `json:"sceneId"`
	Kind    string `json:"kind"` // move|rename|dedup-delete|dedup-maybe|hardlink-unlink|sidecar|symlink|symlink-prune|rmdir
	From    string `json:"from"`
	To      string `json:"to,omitempty"`
	Note    string `json:"note,omitempty"`
}

// Result summarises a run.
type Result struct {
	DryRun         bool     `json:"dryRun"`
	ScenesActed    int      `json:"scenesActed"`
	FilesMoved     int      `json:"filesMoved"`
	FilesRenamed   int      `json:"filesRenamed"`
	Dedups         int      `json:"identicalCopiesDeleted"`
	Hardlinks      int      `json:"hardlinksUnlinked"`
	Sidecars       int      `json:"sidecarsMoved"`
	DirsRemoved    int      `json:"emptyDirsRemoved"`
	Deferred       int      `json:"scenesDeferred"`
	Held           int      `json:"filesHeldRecent"`
	Merged         int      `json:"mergedDuplicateScenes"`
	Symlinks       int      `json:"actorSymlinksCreated"`
	SymlinksPruned int      `json:"actorSymlinksPruned"`
	BytesReclaimed int64    `json:"bytesReclaimed"`
	Actions        []Action `json:"actions"`
}

// ---- sanitisation ----

var reParen = regexp.MustCompile(`\s*\([^)]*\)\s*$`)
var reWS = regexp.MustCompile(`\s+`)
var reNonWord = regexp.MustCompile(`[^\w]`)
var reNonWordDot = regexp.MustCompile(`[^\w\.]`)
var reDoubleDot = regexp.MustCompile(`\.\..+$`)

func sanitizeSite(site string) string {
	site = reParen.ReplaceAllString(site, "")
	site = reWS.ReplaceAllString(site, "")
	return reNonWord.ReplaceAllString(site, "")
}

var reWordSplit = regexp.MustCompile(`[^A-Za-z0-9]+`)

// camelActor renders a performer name as a CamelCase folder name (e.g. "Shalina
// Devine" -> "ShalinaDevine"), capitalising each word and stripping separators.
func camelActor(name string) string {
	name = reParen.ReplaceAllString(name, "")
	var b strings.Builder
	for _, part := range reWordSplit.Split(name, -1) {
		if part == "" {
			continue
		}
		r := []rune(part)
		b.WriteString(strings.ToUpper(string(r[0])))
		if len(r) > 1 {
			b.WriteString(string(r[1:]))
		}
	}
	return b.String()
}

func sanitizeToken(text string) string {
	if text == "" {
		return ""
	}
	text = reWS.ReplaceAllString(text, ".")
	text = reNonWordDot.ReplaceAllString(text, "")
	text = reDoubleDot.ReplaceAllString(text, "")
	return strings.Trim(text, ".")
}

func fovToken(projection string) string {
	if projection == "" {
		return "VR180"
	}
	if v, ok := fovMap[strings.ToLower(strings.TrimSpace(projection))]; ok {
		return v
	}
	cleaned := reNonWord.ReplaceAllString(strings.ToUpper(projection), "")
	if cleaned == "" {
		return "VR180"
	}
	return cleaned
}

func parseReleaseDate(s models.Scene) (time.Time, bool) {
	if s.ReleaseDateText != "" && len(s.ReleaseDateText) >= 10 {
		if t, err := time.Parse("2006-01-02", s.ReleaseDateText[:10]); err == nil {
			return t, true
		}
	}
	if !s.ReleaseDate.IsZero() {
		return s.ReleaseDate, true
	}
	return time.Time{}, false
}

// selectCast picks the performers used for naming: drop "aka:" alias records (unless
// they're all there is), then select by the configured gender preference. With a
// preference of "female" or "male", performers of that gender are used (falling back to
// unknown-gender performers when none match); "any" (or empty) keeps every performer
// regardless of gender. Names are returned raw (unsanitised), capped at maxCast.
func selectCast(cast []models.Actor, pref string) []string {
	// Match the original tool: consider performers in alphabetical order by name so
	// the folder cast order is stable (and doesn't churn already-correct folders).
	sorted := make([]models.Actor, len(cast))
	copy(sorted, cast)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	real := sorted[:0:0]
	for _, a := range sorted {
		if !strings.HasPrefix(strings.ToLower(a.Name), "aka:") {
			real = append(real, a)
		}
	}
	pool := real
	if len(pool) == 0 {
		pool = sorted
	}
	pref = strings.ToLower(strings.TrimSpace(pref))
	var preferred, unknowns []string
	for _, a := range pool {
		g := strings.ToLower(strings.TrimSpace(a.Gender))
		switch pref {
		case "female", "male":
			switch g {
			case pref:
				preferred = append(preferred, a.Name)
			case "female", "male", "non_binary":
				// a specified gender that isn't the preferred one; excluded
			default:
				unknowns = append(unknowns, a.Name)
			}
		default:
			// no preference: keep every performer
			preferred = append(preferred, a.Name)
		}
	}
	chosen := preferred
	if len(chosen) == 0 {
		chosen = unknowns
	}
	if len(chosen) > maxCast {
		chosen = chosen[:maxCast]
	}
	return chosen
}

// castNames returns the selected performers as sanitised folder-name tokens.
func castNames(cast []models.Actor, pref string) []string {
	chosen := selectCast(cast, pref)
	out := make([]string, 0, len(chosen))
	for _, n := range chosen {
		if tok := sanitizeToken(n); tok != "" {
			out = append(out, tok)
		}
	}
	return out
}

func buildDirName(site string, dt time.Time, cast []string, title, fov string, height int) (studio, dir string) {
	studio = sanitizeSite(site)
	segs := []string{studio, dt.Format("06.01.02")}
	if c := strings.Join(cast, "."); c != "" {
		segs = append(segs, c)
	}
	segs = append(segs, title, "XXX", fov, strconv.Itoa(height)+"p")
	var kept []string
	for _, s := range segs {
		if s != "" {
			kept = append(kept, s)
		}
	}
	return studio, strings.Join(kept, ".")
}

// ---- fs helpers ----

func splitExt(name string) (base, ext string) {
	ext = filepath.Ext(name)
	return name[:len(name)-len(ext)], ext
}

func resolveName(used map[string]bool, name string, height int) string {
	if !used[name] {
		return name
	}
	base, ext := splitExt(name)
	cand := base + "." + strconv.Itoa(height) + "p" + ext
	if !used[cand] {
		return cand
	}
	for i := 2; ; i++ {
		cand = base + "." + strconv.Itoa(height) + "p." + strconv.Itoa(i) + ext
		if !used[cand] {
			return cand
		}
	}
}

var inodeFallback uint64

func inodeKey(fi os.FileInfo) [2]uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return [2]uint64{uint64(st.Dev), uint64(st.Ino)}
	}
	// No inode info: give each file a unique key so distinct files are never mistaken
	// for hard links of one another (which would delete all but one).
	return [2]uint64{^uint64(0), atomic.AddUint64(&inodeFallback, 1)}
}

func fileMD5(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", false
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

func moveFile(src, dst string) error {
	// Never overwrite an existing destination (defense-in-depth against a TOCTOU race
	// between planning and applying).
	if src != dst {
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("destination already exists: %s", dst)
		}
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

func isUnder(path, dir string) bool {
	return dir != "" && (path == dir || strings.HasPrefix(path, dir+"/"))
}
