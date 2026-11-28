package transcode

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackendForFallsBackForCPUOnlyCodec(t *testing.T) {
	nvenc := Backend{
		Name:     BackendNVENC,
		Label:    "NVENC",
		Encoders: map[string]string{CodecH264: "h264_nvenc", CodecHEVC: "hevc_nvenc"},
		GPUIndex: 0,
	}
	cpu := Backend{
		Name:     BackendCPU,
		Label:    "CPU (x264)",
		Encoders: map[string]string{CodecH264: "libx264", CodecX264: "libx264"},
	}
	hw := Hardware{Preferred: nvenc, Available: []Backend{nvenc, cpu}}

	if b, ok := hw.BackendFor(CodecHEVC); !ok || b.Name != BackendNVENC {
		t.Errorf("HEVC backend = %q, %v; want NVENC", b.Name, ok)
	}
	if b, ok := hw.BackendFor(CodecX264); !ok || b.Name != BackendCPU {
		t.Errorf("x264 backend = %q, %v; want CPU", b.Name, ok)
	}
	if _, ok := hw.BackendFor(CodecAV1); ok {
		t.Error("AV1 should not be available")
	}
}

func TestVAAPIArgsUploadFrames(t *testing.T) {
	plan := Plan{
		Codec:   CodecH264,
		Encoder: "h264_vaapi",
		Backend: Backend{Name: BackendVAAPI, Device: "/dev/dri/renderD128"},
		Width:   1920,
		Height:  1080,
		FPS:     60,
	}
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 10, Options{BitrateMbps: 20, AudioKbps: 128}, plan, -1), " ")
	for _, want := range []string{
		"-vaapi_device /dev/dri/renderD128",
		"-c:v h264_vaapi",
		"-rc_mode VBR",
		"-vf format=nv12,hwupload",
		"pipe:1",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("VAAPI args missing %q: %s", want, args)
		}
	}
	if strings.Contains(args, "-gpu") {
		t.Errorf("VAAPI args should not select an NVENC GPU: %s", args)
	}
}

func TestQSVArgsUploadFrames(t *testing.T) {
	plan := Plan{
		Codec:   CodecHEVC,
		Encoder: "hevc_qsv",
		Backend: Backend{Name: BackendQSV, Device: "/dev/dri/renderD128"},
		Width:   3840,
		Height:  2160,
		FPS:     60,
		Scale:   true,
	}
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, plan, -1), " ")
	for _, want := range []string{
		"-init_hw_device qsv=hw:/dev/dri/renderD128",
		"-filter_hw_device hw",
		"-c:v hevc_qsv",
		"-vf scale=3840:2160,format=nv12,hwupload=extra_hw_frames=64",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("QSV args missing %q: %s", want, args)
		}
	}
}

func TestTestArgsIsUnpaced(t *testing.T) {
	plan := Plan{
		Codec:   CodecH264,
		Encoder: "h264_nvenc",
		Backend: Backend{Name: BackendNVENC, GPUIndex: 0},
		Width:   1920,
		Height:  1080,
		FPS:     60,
	}
	args := strings.Join(TestArgs("ffmpeg", "/videos/scene.mp4", 2, Options{}, plan, "/tmp/test.mp4"), " ")
	if strings.Contains(args, "-re") {
		t.Errorf("test encode should not be paced: %s", args)
	}
	for _, want := range []string{"-t 2.000", "-progress pipe:1", "-an", "/tmp/test.mp4"} {
		if !strings.Contains(args, want) {
			t.Errorf("test args missing %q: %s", want, args)
		}
	}
}

// fakeProbe replaces the hardware probe with one that takes delay and labels each result
// with how many probes have run.
func fakeProbe(t *testing.T, delay time.Duration) *int32 {
	t.Helper()
	var calls int32
	saved := probe
	probe = func(ffmpegPath string) Hardware {
		n := atomic.AddInt32(&calls, 1)
		time.Sleep(delay)
		return Hardware{FFmpeg: ffmpegPath, Notes: []string{fmt.Sprintf("probe %d", n)}}
	}
	resetHardwareCache()
	t.Cleanup(func() {
		resetHardwareCache()
		probe = saved
	})
	return &calls
}

// resetHardwareCache waits for a running probe, then forgets every result.
func resetHardwareCache() {
	hardwareCache.Lock()
	run := hardwareCache.running
	hardwareCache.Unlock()
	if run != nil {
		<-run.done
	}
	hardwareCache.Lock()
	hardwareCache.ffmpeg, hardwareCache.hw, hardwareCache.ok, hardwareCache.running = "", Hardware{}, false, nil
	hardwareCache.Unlock()
}

func waitForProbe(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for Detect("/bin/ffmpeg").Notes[0] != want {
		if time.Now().After(deadline) {
			t.Fatalf("Detect never returned %s", want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestDetectProbesOnceAndCaches(t *testing.T) {
	calls := fakeProbe(t, 0)
	first := Detect("/bin/ffmpeg")
	second := Detect("/bin/ffmpeg")
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("probes = %d, want 1", got)
	}
	if first.Notes[0] != "probe 1" || second.Notes[0] != "probe 1" {
		t.Fatalf("results = %v, %v; want both from probe 1", first.Notes, second.Notes)
	}
}

func TestReprobeAnswersFromPreviousResultMeanwhile(t *testing.T) {
	calls := fakeProbe(t, 300*time.Millisecond)
	Detect("/bin/ffmpeg")
	Reprobe("/bin/ffmpeg")

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if hw := Detect("/bin/ffmpeg"); hw.Notes[0] != "probe 1" {
				t.Errorf("call during reprobe returned %v, want the previous result", hw.Notes)
			}
		}()
	}
	wg.Wait()
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("calls during reprobe took %v; they waited for the probe", elapsed)
	}
	waitForProbe(t, "probe 2")
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("probes = %d, want 2", got)
	}
}

func TestDetectJoinsStartupProbe(t *testing.T) {
	calls := fakeProbe(t, 200*time.Millisecond)
	Reprobe("/bin/ffmpeg")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if hw := Detect("/bin/ffmpeg"); hw.Notes[0] != "probe 1" {
				t.Errorf("Detect returned %v, want the startup probe's result", hw.Notes)
			}
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("probes = %d, want 1: Detect should join the running probe", got)
	}
}

func TestDetectFreshJoinsRunningProbe(t *testing.T) {
	calls := fakeProbe(t, 200*time.Millisecond)
	Reprobe("/bin/ffmpeg")
	time.Sleep(20 * time.Millisecond)
	if hw := DetectFresh("/bin/ffmpeg"); hw.Notes[0] != "probe 1" {
		t.Fatalf("DetectFresh = %v, want the running probe's result", hw.Notes)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("probes = %d, want 1", got)
	}
}

func TestDetectFreshProbesAgain(t *testing.T) {
	calls := fakeProbe(t, 0)
	Detect("/bin/ffmpeg")
	if hw := DetectFresh("/bin/ffmpeg"); hw.Notes[0] != "probe 2" {
		t.Fatalf("DetectFresh = %v, want a new probe", hw.Notes)
	}
	if hw := Detect("/bin/ffmpeg"); hw.Notes[0] != "probe 2" {
		t.Fatalf("Detect after DetectFresh = %v, want its result", hw.Notes)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("probes = %d, want 2", got)
	}
}

func TestDetectProbesAgainForAnotherFFmpeg(t *testing.T) {
	calls := fakeProbe(t, 0)
	Detect("/bin/ffmpeg")
	if hw := Detect("/opt/ffmpeg"); hw.FFmpeg != "/opt/ffmpeg" {
		t.Fatalf("FFmpeg = %q, want the newly requested binary", hw.FFmpeg)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("probes = %d, want 2", got)
	}
}

func TestProbeErrorNoteExplainsOldLibva(t *testing.T) {
	abort := fmt.Errorf("signal: aborted: implib-gen: libva.so.2: failed to resolve symbol 'vaMapBuffer2' via dlsym: /lib/x86_64-linux-gnu/libva.so.2: undefined symbol: vaMapBuffer2; ffmpeg: libva.so.2.init.c:290: _libva_so_2_tramp_resolve: Assertion failed.")
	if got := probeErrorNote(abort); !strings.Contains(got, "libva 2.21") {
		t.Errorf("note %q should name the libva version needed", got)
	}
	other := fmt.Errorf("exit status 1: Error while opening encoder")
	if got := probeErrorNote(other); got != compactError(other) {
		t.Errorf("other errors should be passed through, got %q", got)
	}
}
