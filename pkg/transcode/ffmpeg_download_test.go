package transcode

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ulikunitz/xz"
)

const fakeFFmpeg = "#!/bin/sh\necho ffmpeg version test\n"

func tarXZ(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(xw)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type fakeRelease struct {
	requests atomic.Int32
	release  chan struct{} // when set, downloads wait for it to close
	entered  chan struct{} // receives once per download that has started
}

// serveRelease points the downloader at a test server that serves archive for platform, and
// returns a fresh bin folder.
func serveRelease(t *testing.T, platform, name string, archive []byte, checksum string) (*fakeRelease, string) {
	t.Helper()
	rel := &fakeRelease{entered: make(chan struct{}, 8)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel.requests.Add(1)
		rel.entered <- struct{}{}
		if rel.release != nil {
			<-rel.release
		}
		if !strings.HasSuffix(r.URL.Path, "/v"+JellyfinFFmpegVersion+"/"+name) {
			http.NotFound(w, r)
			return
		}
		w.Write(archive)
	}))
	t.Cleanup(srv.Close)

	savedPlatform, savedURL, savedAssets := jellyfinPlatform, jellyfinDownloadURL, jellyfinAssets
	savedRuns, savedCandidates := checkFFmpegRuns, vaapiDriverCandidates
	t.Cleanup(func() {
		jellyfinPlatform, jellyfinDownloadURL, jellyfinAssets = savedPlatform, savedURL, savedAssets
		checkFFmpegRuns, vaapiDriverCandidates = savedRuns, savedCandidates
		managedFFmpeg.Lock()
		managedFFmpeg.running, managedFFmpeg.err = nil, nil
		managedFFmpeg.downloaded, managedFFmpeg.total = 0, 0
		managedFFmpeg.Unlock()
	})
	jellyfinPlatform = platform
	jellyfinDownloadURL = srv.URL + "/v%s/%s"
	jellyfinAssets = map[string]jellyfinAsset{platform: {name: name, sha256: checksum}}
	checkFFmpegRuns = func(string) error { return nil }
	vaapiDriverCandidates = nil
	return rel, t.TempDir()
}

func TestEnsureManagedFFmpegInstallsFromTarXZ(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": fakeFFmpeg, "ffprobe": "probe"})
	rel, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, sha256Hex(archive))

	if s := ManagedFFmpegStatus(bin); s.State != FFmpegMissing {
		t.Fatalf("state before download = %q, want %q", s.State, FFmpegMissing)
	}
	path, err := EnsureManagedFFmpeg(bin)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(bin, "jellyfin-ffmpeg-"+JellyfinFFmpegVersion, "ffmpeg"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != fakeFFmpeg {
		t.Fatalf("installed ffmpeg = %q, %v", got, err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed ffmpeg is not executable: %v", info.Mode())
	}
	if info, _ := os.Stat(filepath.Dir(path)); info.Mode().Perm() != 0o755 {
		t.Fatalf("install folder mode = %v, want 0755", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "ffprobe")); !os.IsNotExist(err) {
		t.Fatalf("only ffmpeg should be installed, ffprobe stat err = %v", err)
	}
	if s := ManagedFFmpegStatus(bin); s.State != FFmpegReady || s.Path != path {
		t.Fatalf("status = %+v, want ready at %s", s, path)
	}

	// Installed: no second download.
	if again, err := EnsureManagedFFmpeg(bin); err != nil || again != path {
		t.Fatalf("second call = %q, %v", again, err)
	}
	if n := rel.requests.Load(); n != 1 {
		t.Fatalf("downloads = %d, want 1", n)
	}
	assertNoTempFiles(t, bin)
}

func TestEnsureManagedFFmpegInstallsFromZipOnWindows(t *testing.T) {
	archive := zipArchive(t, map[string]string{"ffmpeg.exe": "MZ", "ffprobe.exe": "MZ"})
	_, bin := serveRelease(t, "windows/amd64", "ffmpeg-win.zip", archive, sha256Hex(archive))

	path, err := EnsureManagedFFmpeg(bin)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "ffmpeg.exe" {
		t.Fatalf("path = %q, want ffmpeg.exe", path)
	}
	if got, _ := os.ReadFile(path); string(got) != "MZ" {
		t.Fatalf("installed ffmpeg.exe = %q", got)
	}
}

func TestEnsureManagedFFmpegRejectsWrongChecksum(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": fakeFFmpeg})
	_, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, strings.Repeat("0", 64))

	if _, err := EnsureManagedFFmpeg(bin); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("err = %v, want a checksum error", err)
	}
	if _, ok := ManagedFFmpeg(bin); ok {
		t.Fatal("ffmpeg installed despite the checksum mismatch")
	}
	if s := ManagedFFmpegStatus(bin); s.State != FFmpegFailed || !strings.Contains(s.Error, "checksum") {
		t.Fatalf("status = %+v, want failed with the checksum error", s)
	}
	assertNoTempFiles(t, bin)
}

func TestEnsureManagedFFmpegRejectsBinaryThatDoesNotRun(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": fakeFFmpeg})
	_, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, sha256Hex(archive))
	checkFFmpegRuns = func(string) error { return errors.New("exec format error") }

	if _, err := EnsureManagedFFmpeg(bin); err == nil || !strings.Contains(err.Error(), "does not run") {
		t.Fatalf("err = %v, want a does-not-run error", err)
	}
	if _, ok := ManagedFFmpeg(bin); ok {
		t.Fatal("ffmpeg installed although it does not run")
	}
	assertNoTempFiles(t, bin)
}

func TestEnsureManagedFFmpegReportsMissingBinary(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffprobe": "probe"})
	_, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, sha256Hex(archive))

	if _, err := EnsureManagedFFmpeg(bin); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want ffmpeg not found", err)
	}
}

func TestManagedFFmpegUnsupportedPlatform(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": fakeFFmpeg})
	_, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, sha256Hex(archive))
	jellyfinPlatform = "linux/arm"

	if _, ok := ManagedFFmpegPath(bin); ok {
		t.Fatal("linux/arm should have no build")
	}
	if s := ManagedFFmpegStatus(bin); s.State != FFmpegUnsupported {
		t.Fatalf("state = %q, want %q", s.State, FFmpegUnsupported)
	}
	if _, err := EnsureManagedFFmpeg(bin); err == nil {
		t.Fatal("expected an error for an unsupported platform")
	}
}

func TestEnsureManagedFFmpegSharesOneDownload(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": fakeFFmpeg})
	rel, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, sha256Hex(archive))
	rel.release = make(chan struct{})

	const callers = 4
	var wg sync.WaitGroup
	paths := make([]string, callers)
	errs := make([]error, callers)
	wg.Add(1)
	go func() { defer wg.Done(); paths[0], errs[0] = EnsureManagedFFmpeg(bin) }()
	<-rel.entered
	if !ManagedFFmpegDownloading() {
		t.Fatal("download should be in progress")
	}
	if s := ManagedFFmpegStatus(bin); s.State != FFmpegDownloading {
		t.Fatalf("state = %q, want %q", s.State, FFmpegDownloading)
	}
	for i := 1; i < callers; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); paths[i], errs[i] = EnsureManagedFFmpeg(bin) }(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(rel.release)
	wg.Wait()

	for i := range paths {
		if errs[i] != nil || paths[i] == "" || paths[i] != paths[0] {
			t.Fatalf("caller %d = %q, %v; want %q", i, paths[i], errs[i], paths[0])
		}
	}
	if n := rel.requests.Load(); n != 1 {
		t.Fatalf("downloads = %d, want 1", n)
	}
	if ManagedFFmpegDownloading() {
		t.Fatal("download should have finished")
	}
}

func TestEnsureManagedFFmpegRemovesOlderVersions(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": fakeFFmpeg})
	_, bin := serveRelease(t, "linux/amd64", "ffmpeg-linux.tar.xz", archive, sha256Hex(archive))
	old := filepath.Join(bin, "jellyfin-ffmpeg-7.0.0-1")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(bin, "ffmpeg")
	if err := os.WriteFile(other, []byte("preview ffmpeg"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureManagedFFmpeg(bin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old version still present: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("XBVR's own ffmpeg was removed: %v", err)
	}
}

func TestVAAPIDriverDirs(t *testing.T) {
	root := t.TempDir()
	withDriver := filepath.Join(root, "lib64", "dri")
	empty := filepath.Join(root, "lib", "dri")
	for _, dir := range []string{withDriver, empty} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(withDriver, "iHD_drv_video.so"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got := vaapiDriverDirs([]string{empty, withDriver, filepath.Join(root, "missing")})
	if len(got) != 1 || got[0] != withDriver {
		t.Fatalf("vaapiDriverDirs = %v, want [%s]", got, withDriver)
	}
}

func TestJellyfinAssetsCoverReleasePlatforms(t *testing.T) {
	for _, platform := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64"} {
		asset, ok := jellyfinAssets[platform]
		if !ok {
			t.Errorf("no jellyfin-ffmpeg build for %s", platform)
			continue
		}
		if !strings.Contains(asset.name, JellyfinFFmpegVersion) || len(asset.sha256) != 64 {
			t.Errorf("%s: asset %+v does not match version %s", platform, asset, JellyfinFFmpegVersion)
		}
	}
}

func assertNoTempFiles(t *testing.T, bin string) {
	t.Helper()
	entries, _ := os.ReadDir(bin)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".jellyfin-ffmpeg-") {
			t.Errorf("temporary file left behind: %s", e.Name())
		}
	}
}
