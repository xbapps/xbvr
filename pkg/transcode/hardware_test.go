package transcode

import (
	"strings"
	"testing"
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
