package transcode

import (
	"archive/tar"
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xi2/xz"
)

// JellyfinFFmpegVersion is the jellyfin-ffmpeg release downloaded for live transcoding. Its
// portable builds include NVENC, Quick Sync, VAAPI and x264, and need only the C library and
// the GPU drivers already on the system. On Linux, VAAPI and Quick Sync also need libva 2.21
// or newer to copy frames to the GPU.
const JellyfinFFmpegVersion = "8.1.3-2"

type jellyfinAsset struct {
	name   string
	sha256 string
}

// jellyfinAssets maps GOOS/GOARCH to the portable build for that platform.
var jellyfinAssets = map[string]jellyfinAsset{
	"linux/amd64":   {"jellyfin-ffmpeg_8.1.3-2_portable_linux64-gpl.tar.xz", "1df84d04c4322902dba81c290508cdf3d89524d210b9fe03154bea6c4e3bfbb6"},
	"linux/arm64":   {"jellyfin-ffmpeg_8.1.3-2_portable_linuxarm64-gpl.tar.xz", "6522585818059cc886363fb1dee86ac9b8579fec3b914337fb92974ef9eb9884"},
	"darwin/amd64":  {"jellyfin-ffmpeg_8.1.3-2_portable_mac64-gpl.tar.xz", "d394180f1155f84857544a48a99869f5741ccc50c5a1f73d2797429a2b326448"},
	"darwin/arm64":  {"jellyfin-ffmpeg_8.1.3-2_portable_macarm64-gpl.tar.xz", "d898cc819e7c4e9219d6410ddbdaada0838f14bec6ff89e283639915394cb43f"},
	"windows/amd64": {"jellyfin-ffmpeg_8.1.3-2_portable_win64-clang-gpl.zip", "b65dd5011a80b5abec08bd559489cf188217238c8e38b5bf553504c974f5694d"},
	"windows/arm64": {"jellyfin-ffmpeg_8.1.3-2_portable_winarm64-clang-gpl.zip", "e1e0373cb989440590ce2d124f40988e3165ed338971ae14a97476cd61284785"},
}

var (
	jellyfinPlatform    = runtime.GOOS + "/" + runtime.GOARCH
	jellyfinDownloadURL = "https://github.com/jellyfin/jellyfin-ffmpeg/releases/download/v%s/%s"
	jellyfinHTTPClient  = &http.Client{Timeout: 30 * time.Minute}
	checkFFmpegRuns     = ffmpegRuns
)

const (
	jellyfinDirPrefix = "jellyfin-ffmpeg-"
	// maxFFmpegBytes bounds the extracted binary; the largest build is about 110 MB.
	maxFFmpegBytes = 512 << 20
)

// FFmpeg download states reported by ManagedFFmpegStatus.
const (
	FFmpegReady       = "ready"
	FFmpegDownloading = "downloading"
	FFmpegFailed      = "failed"
	FFmpegMissing     = "missing"
	FFmpegUnsupported = "unsupported"
)

// FFmpegStatus describes the downloaded ffmpeg for the settings page.
type FFmpegStatus struct {
	Version    string `json:"version"`
	Platform   string `json:"platform"`
	State      string `json:"state"`
	Path       string `json:"path,omitempty"`
	Error      string `json:"error,omitempty"`
	Downloaded int64  `json:"downloaded,omitempty"`
	Total      int64  `json:"total,omitempty"`
}

// managedFFmpeg tracks the download. At most one runs at a time; callers that arrive while it
// runs wait for it.
var managedFFmpeg struct {
	sync.Mutex
	running    chan struct{}
	err        error
	downloaded int64
	total      int64
}

func jellyfinExe(name string) string {
	if strings.HasPrefix(jellyfinPlatform, "windows/") {
		return name + ".exe"
	}
	return name
}

// ManagedFFmpegPath returns where the downloaded ffmpeg lives under binDir, and false when
// jellyfin-ffmpeg has no build for this platform.
func ManagedFFmpegPath(binDir string) (string, bool) {
	if _, ok := jellyfinAssets[jellyfinPlatform]; !ok {
		return "", false
	}
	return filepath.Join(binDir, jellyfinDirPrefix+JellyfinFFmpegVersion, jellyfinExe("ffmpeg")), true
}

// ManagedFFmpeg returns the downloaded ffmpeg if it is installed.
func ManagedFFmpeg(binDir string) (string, bool) {
	path, ok := ManagedFFmpegPath(binDir)
	if !ok {
		return "", false
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return "", false
	}
	setVAAPIDriverPath()
	return path, true
}

// ManagedFFmpegDownloading reports whether a download is in progress.
func ManagedFFmpegDownloading() bool {
	managedFFmpeg.Lock()
	defer managedFFmpeg.Unlock()
	return managedFFmpeg.running != nil
}

// ManagedFFmpegStatus reports whether the downloaded ffmpeg is installed, downloading or failed.
func ManagedFFmpegStatus(binDir string) FFmpegStatus {
	s := FFmpegStatus{Version: JellyfinFFmpegVersion, Platform: jellyfinPlatform}
	if _, ok := ManagedFFmpegPath(binDir); !ok {
		s.State = FFmpegUnsupported
		return s
	}
	managedFFmpeg.Lock()
	running, err := managedFFmpeg.running != nil, managedFFmpeg.err
	s.Downloaded, s.Total = managedFFmpeg.downloaded, managedFFmpeg.total
	managedFFmpeg.Unlock()
	switch path, ok := ManagedFFmpeg(binDir); {
	case running:
		s.State = FFmpegDownloading
	case ok:
		s.State, s.Path, s.Downloaded, s.Total = FFmpegReady, path, 0, 0
	case err != nil:
		s.State, s.Error = FFmpegFailed, err.Error()
	default:
		s.State = FFmpegMissing
	}
	return s
}

// EnsureManagedFFmpeg downloads and installs jellyfin-ffmpeg for this platform unless it is
// already installed, and returns its path. Callers that arrive during a download share it.
func EnsureManagedFFmpeg(binDir string) (string, error) {
	if path, ok := ManagedFFmpeg(binDir); ok {
		return path, nil
	}
	asset, ok := jellyfinAssets[jellyfinPlatform]
	if !ok {
		return "", fmt.Errorf("jellyfin-ffmpeg has no build for %s", jellyfinPlatform)
	}

	managedFFmpeg.Lock()
	if run := managedFFmpeg.running; run != nil {
		managedFFmpeg.Unlock()
		<-run
		managedFFmpeg.Lock()
		err := managedFFmpeg.err
		managedFFmpeg.Unlock()
		if path, ok := ManagedFFmpeg(binDir); ok {
			return path, nil
		}
		return "", err
	}
	run := make(chan struct{})
	managedFFmpeg.running, managedFFmpeg.err = run, nil
	managedFFmpeg.downloaded, managedFFmpeg.total = 0, 0
	managedFFmpeg.Unlock()

	started := time.Now()
	common.Log.Infof("transcode: downloading jellyfin-ffmpeg %s for %s", JellyfinFFmpegVersion, jellyfinPlatform)
	path, _ := ManagedFFmpegPath(binDir)
	err := installJellyfinFFmpeg(binDir, filepath.Dir(path), asset)
	if err != nil {
		common.Log.Warnf("transcode: jellyfin-ffmpeg download failed: %v", err)
	} else {
		common.Log.Infof("transcode: installed jellyfin-ffmpeg %s in %v", JellyfinFFmpegVersion, time.Since(started).Round(time.Second))
	}

	managedFFmpeg.Lock()
	managedFFmpeg.err = err
	managedFFmpeg.running = nil
	managedFFmpeg.Unlock()
	close(run)

	if err != nil {
		return "", err
	}
	setVAAPIDriverPath()
	return path, nil
}

// installJellyfinFFmpeg downloads the archive, checks its SHA-256, extracts ffmpeg into a
// temporary folder, checks that it runs, then moves the folder to final and removes older
// versions.
func installJellyfinFFmpeg(binDir, final string, asset jellyfinAsset) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	archive, err := os.CreateTemp(binDir, ".jellyfin-ffmpeg-*.download")
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name())
	defer archive.Close()

	if err := downloadJellyfinAsset(archive, asset); err != nil {
		return err
	}

	staging, err := os.MkdirTemp(binDir, ".jellyfin-ffmpeg-*.tmp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	// MkdirTemp creates the folder private to its owner; the installed folder is readable
	// like the rest of the bin folder.
	if err := os.Chmod(staging, 0o755); err != nil {
		return err
	}

	exe := filepath.Join(staging, jellyfinExe("ffmpeg"))
	if err := extractFFmpeg(archive, asset.name, exe); err != nil {
		return err
	}
	if err := checkFFmpegRuns(exe); err != nil {
		return fmt.Errorf("the downloaded ffmpeg does not run on this system: %w", err)
	}

	_ = os.RemoveAll(final)
	if err := os.Rename(staging, final); err != nil {
		return err
	}
	removeOldJellyfinFFmpeg(binDir, filepath.Base(final))
	return nil
}

func downloadJellyfinAsset(dst *os.File, asset jellyfinAsset) error {
	url := fmt.Sprintf(jellyfinDownloadURL, JellyfinFFmpegVersion, asset.name)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "xbvr")
	resp, err := jellyfinHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}

	managedFFmpeg.Lock()
	managedFFmpeg.total = resp.ContentLength
	managedFFmpeg.Unlock()

	hash := sha256.New()
	progress := &progressWriter{total: resp.ContentLength}
	if _, err := io.Copy(io.MultiWriter(dst, hash, progress), resp.Body); err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != asset.sha256 {
		return fmt.Errorf("download %s: checksum %s does not match the expected %s", asset.name, got, asset.sha256)
	}
	return nil
}

// progressWriter records the bytes downloaded for the status, and logs every quarter.
type progressWriter struct {
	total, done int64
	logged      int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	managedFFmpeg.Lock()
	managedFFmpeg.downloaded = p.done
	managedFFmpeg.Unlock()
	if p.total > 0 {
		if quarter := p.done * 4 / p.total; quarter > p.logged && quarter < 4 {
			p.logged = quarter
			common.Log.Infof("transcode: jellyfin-ffmpeg download %d%%", quarter*25)
		}
	}
	return len(b), nil
}

// extractFFmpeg copies the ffmpeg binary out of a .tar.xz or .zip archive to dst.
func extractFFmpeg(archive *os.File, name, dst string) error {
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	want := jellyfinExe("ffmpeg")
	switch {
	case strings.HasSuffix(name, ".tar.xz"):
		xr, err := xz.NewReader(archive, 0)
		if err != nil {
			return err
		}
		tr := tar.NewReader(xr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == want {
				return writeExecutable(dst, tr)
			}
		}
	case strings.HasSuffix(name, ".zip"):
		info, err := archive.Stat()
		if err != nil {
			return err
		}
		zr, err := zip.NewReader(archive, info.Size())
		if err != nil {
			return err
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() || filepath.Base(f.Name) != want {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			return writeExecutable(dst, rc)
		}
	default:
		return fmt.Errorf("unknown archive type: %s", name)
	}
	return fmt.Errorf("%s not found in %s", want, name)
}

func writeExecutable(dst string, r io.Reader) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxFFmpegBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxFFmpegBytes {
		err = errors.New("ffmpeg in the archive is larger than expected")
	}
	return err
}

func ffmpegRuns(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "-hide_banner", "-version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, compactError(errors.New(strings.TrimSpace(string(out)))))
	}
	return nil
}

func removeOldJellyfinFFmpeg(binDir, keep string) {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), jellyfinDirPrefix) && e.Name() != keep {
			if err := os.RemoveAll(filepath.Join(binDir, e.Name())); err == nil {
				common.Log.Infof("transcode: removed old %s", e.Name())
			}
		}
	}
}

var vaapiDriverPathOnce sync.Once

// setVAAPIDriverPath points libva at the system's VAAPI drivers. The portable Linux builds
// look only in Debian's driver folder, so on other distributions LIBVA_DRIVERS_PATH is set to
// the folders that hold drivers, unless it is already set.
func setVAAPIDriverPath() {
	vaapiDriverPathOnce.Do(func() {
		if !strings.HasPrefix(jellyfinPlatform, "linux/") || os.Getenv("LIBVA_DRIVERS_PATH") != "" {
			return
		}
		if dirs := vaapiDriverDirs(vaapiDriverCandidates); len(dirs) > 0 {
			os.Setenv("LIBVA_DRIVERS_PATH", strings.Join(dirs, ":"))
		}
	})
}

var vaapiDriverCandidates = []string{
	"/usr/lib/x86_64-linux-gnu/dri",
	"/usr/lib/aarch64-linux-gnu/dri",
	"/usr/lib64/dri",
	"/usr/lib/dri",
	"/usr/local/lib/dri",
	"/usr/local/lib64/dri",
	"/run/opengl-driver/lib/dri",
}

// vaapiDriverDirs returns the candidate folders that contain a VAAPI driver.
func vaapiDriverDirs(candidates []string) []string {
	var dirs []string
	for _, dir := range candidates {
		if matches, _ := filepath.Glob(filepath.Join(dir, "*_drv_video.so")); len(matches) > 0 {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}
