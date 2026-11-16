package transcode

import (
	"strings"
	"testing"
)

func TestParseCodecs(t *testing.T) {
	got := ParseCodecs([]string{"H264", "hevc", "av1", "h264", "bogus", ""})
	if strings.Join(got, ",") != "h264,hevc,av1" {
		t.Errorf("ParseCodecs = %v", got)
	}
	if got := ParseCodecs(nil); len(got) != 0 {
		t.Errorf("empty list = %v", got)
	}
}

func TestCodecLabel(t *testing.T) {
	cases := map[string]string{
		CodecH264: "H.264 (NVENC)",
		CodecHEVC: "HEVC (NVENC)",
		CodecAV1:  "AV1 (NVENC)",
		CodecX264: "H.264 (x264)",
	}
	for codec, want := range cases {
		if got := CodecLabel(codec); got != want {
			t.Errorf("CodecLabel(%q) = %q, want %q", codec, got, want)
		}
	}
}

func TestPlanScalesForH264Limit(t *testing.T) {
	p := PlanFor(CodecH264, Options{}, 7680, 3840, 60, 0)
	if !p.Scale || p.Width != 4096 || p.Height != 2048 {
		t.Errorf("H.264 plan = %dx%d scale=%v, want 4096x2048 scaled", p.Width, p.Height, p.Scale)
	}
	if p.FPS != 0 {
		t.Errorf("60 fps should be kept after scaling, got %v", p.FPS)
	}
	if len(p.Notes) == 0 || !strings.Contains(strings.Join(p.Notes, " "), "scaled") {
		t.Errorf("notes should say the source was scaled: %v", p.Notes)
	}
}

func TestPlanKeepsEightKForHEVC(t *testing.T) {
	p := PlanFor(CodecHEVC, Options{}, 7680, 3840, 60, 0)
	if p.Scale || p.Width != 7680 || p.Height != 3840 {
		t.Errorf("HEVC plan = %dx%d scale=%v", p.Width, p.Height, p.Scale)
	}
	if p.FPS != 0 {
		t.Errorf("HEVC should keep 60 fps at 7680x3840, got %v", p.FPS)
	}
}

func TestPlanDropsToThirtyFPSWhenTooLarge(t *testing.T) {
	p := PlanFor(CodecHEVC, Options{}, 8192, 8192, 60, 0)
	if p.FPS != 30 {
		t.Errorf("8192x8192 at 60 fps should drop to 30, got %v", p.FPS)
	}
	if !strings.Contains(strings.Join(p.Notes, " "), "using 30 fps") {
		t.Errorf("notes should explain the frame rate cap: %v", p.Notes)
	}
}

func TestPlanCapsSourceFPSAtSixty(t *testing.T) {
	p := PlanFor(CodecAV1, Options{}, 1920, 1080, 90, 0)
	if p.FPS != 60 {
		t.Errorf("90 fps should be limited to 60, got %v", p.FPS)
	}
}

func TestPlanHonoursHeightCap(t *testing.T) {
	p := PlanFor(CodecH264, Options{MaxHeight: 1080}, 3840, 1920, 30, 0)
	if p.Width != 2160 || p.Height != 1080 || !p.Scale {
		t.Errorf("height cap plan = %dx%d scale=%v, want 2160x1080", p.Width, p.Height, p.Scale)
	}
}

func TestVirtualSize(t *testing.T) {
	o := Options{BitrateMbps: 20, AudioKbps: 128}
	// 60 s at 20 Mbps + 128 kbps = 150,960,000 bytes.
	if got := o.VirtualSize(60); got != 150_960_000 {
		t.Errorf("VirtualSize(60) = %d, want 150960000", got)
	}
	if got := o.VirtualSize(0); got != 0 {
		t.Errorf("VirtualSize(0) = %d, want 0", got)
	}
}

func TestArgsCodecs(t *testing.T) {
	opts := Options{BitrateMbps: 20, AudioKbps: 128}
	cases := map[string]string{
		CodecH264: "h264_nvenc",
		CodecHEVC: "hevc_nvenc",
		CodecAV1:  "av1_nvenc",
		CodecX264: "libx264",
	}
	for codec, encoder := range cases {
		plan := PlanFor(codec, opts, 1920, 1080, 60, 0)
		args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 42.5, opts, plan, 0), " ")
		for _, want := range []string{"-ss 42.500", "-re", encoder, "-b:v 20000k", "-b:a 128k", "pipe:1"} {
			if !strings.Contains(args, want) {
				t.Errorf("%s args missing %q: %s", codec, want, args)
			}
		}
	}
}

func TestPlanScalesForFreeVRAM(t *testing.T) {
	p := PlanFor(CodecAV1, Options{}, 7680, 3840, 60, 1900)
	if p.Unavailable {
		t.Fatalf("4K should fit in 1900 MiB: %v", p.Notes)
	}
	if p.Height != 2160 || p.Width != 4320 {
		t.Errorf("AV1 8K at 1900 MiB free should scale to 4320x2160, got %dx%d", p.Width, p.Height)
	}
	if !strings.Contains(strings.Join(p.Notes, " "), "free VRAM") {
		t.Errorf("notes should explain the VRAM scaling: %v", p.Notes)
	}
}

func TestPlanHEVC8KFitsInNineteenHundredMiB(t *testing.T) {
	p := PlanFor(CodecHEVC, Options{}, 7680, 3840, 60, 1900)
	if p.Unavailable || p.Scale {
		t.Errorf("HEVC 8K should fit in 1900 MiB: %dx%d %v", p.Width, p.Height, p.Notes)
	}
}

func TestPlanUnavailableWhenVRAMIsTooLow(t *testing.T) {
	p := PlanFor(CodecAV1, Options{}, 7680, 3840, 60, 400)
	if !p.Unavailable {
		t.Errorf("400 MiB should not fit any useful AV1 stream: %dx%d %v", p.Width, p.Height, p.Notes)
	}
}

func TestArgsSelectsGPU(t *testing.T) {
	plan := PlanFor(CodecHEVC, Options{}, 1920, 1080, 60, 0)
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, plan, 1), " ")
	if !strings.Contains(args, "-gpu 1") {
		t.Errorf("GPU selection missing: %s", args)
	}
}

func TestArgsScalesAndCapsFPS(t *testing.T) {
	plan := PlanFor(CodecH264, Options{}, 7680, 3840, 60, 0)
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, plan, 0), " ")
	if !strings.Contains(args, "scale=4096:2048") {
		t.Errorf("scale filter missing: %s", args)
	}

	plan = PlanFor(CodecHEVC, Options{}, 8192, 8192, 60, 0)
	args = strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, plan, 0), " ")
	if !strings.Contains(args, "-r 30.000") {
		t.Errorf("frame rate cap missing: %s", args)
	}
	if strings.Contains(args, "scale=") {
		t.Errorf("HEVC should not scale 8K: %s", args)
	}
}
