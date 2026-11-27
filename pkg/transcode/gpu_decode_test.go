package transcode

import (
	"strings"
	"testing"
)

func gpuDecodeHardware(freeMiB int, convert bool) Hardware {
	b := Backend{
		Name:         BackendNVENC,
		Label:        "NVENC",
		Encoders:     map[string]string{CodecH264: "h264_nvenc", CodecHEVC: "hevc_nvenc", CodecAV1: "av1_nvenc"},
		GPUIndex:     1,
		FreeVRAMMiB:  freeMiB,
		Decoders:     map[string]string{"h264": "h264_cuvid", "hevc": "hevc_cuvid"},
		ConvertOnGPU: convert,
	}
	cpu := Backend{Name: BackendCPU, Label: "CPU (x264)", Encoders: map[string]string{CodecH264: "libx264", CodecX264: "libx264"}}
	return Hardware{Preferred: b, Available: []Backend{b, cpu}, InitialBurst: true}
}

func planGPUDecode(codec string, hw Hardware, src SourceVideo) Plan {
	return PlanForEncoder(codec, hw, Options{}, src.Width, src.Height, 60).WithGPUDecode(src)
}

func TestGPUDecodeChosenForEightBitHEVC(t *testing.T) {
	p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, false), SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"})
	if p.Decoder != "hevc_cuvid" {
		t.Fatalf("decoder = %q, want hevc_cuvid: %v", p.Decoder, p.Notes)
	}
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 30, Options{}, p, -1), " ")
	for _, want := range []string{"-xerror -hwaccel cuda -hwaccel_device 1 -hwaccel_output_format cuda -c:v hevc_cuvid", "-c:v hevc_nvenc"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %s", want, args)
		}
	}
	if strings.Index(args, "hevc_cuvid") > strings.Index(args, "-i /videos") {
		t.Errorf("decoder must be set before the input: %s", args)
	}
	for _, unwanted := range []string{"-pix_fmt", "-vf", "-resize"} {
		if strings.Contains(args, unwanted) {
			t.Errorf("GPU frames need no %s here: %s", unwanted, args)
		}
	}
}

func TestGPUDecodeResizesInTheDecoder(t *testing.T) {
	p := planGPUDecode(CodecH264, gpuDecodeHardware(0, true), SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p10le"})
	if p.Decoder != "hevc_cuvid" {
		t.Fatalf("decoder = %q: %v", p.Decoder, p.Notes)
	}
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, p, -1), " ")
	for _, want := range []string{"-resize 4096x2048", "-vf scale_cuda=format=nv12"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %s", want, args)
		}
	}
	if strings.Contains(args, "scale=4096") || strings.Contains(args, "-pix_fmt") {
		t.Errorf("CPU scaling left in a GPU decode: %s", args)
	}
}

func TestCPUDecodeRestoresCPUFilters(t *testing.T) {
	p := planGPUDecode(CodecH264, gpuDecodeHardware(0, true), SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}).CPUDecode()
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, p, -1), " ")
	if strings.Contains(args, "cuvid") || strings.Contains(args, "-hwaccel") || strings.Contains(args, "-resize") || strings.Contains(args, "-xerror") {
		t.Errorf("CPU decode still selects the GPU decoder: %s", args)
	}
	for _, want := range []string{"scale=4096:2048", "-pix_fmt yuv420p"} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %s", want, args)
		}
	}
}

func TestGPUDecodeNeedsConversionForTenBit(t *testing.T) {
	src := SourceVideo{Codec: "hevc", Width: 5760, Height: 2880, PixFmt: "yuv420p10le"}
	if p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, false), src); p.Decoder != "" {
		t.Errorf("10-bit source decoded on the GPU without scale_cuda: %q", p.Decoder)
	}
	if p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, true), src); p.Decoder != "hevc_cuvid" {
		t.Errorf("10-bit source with scale_cuda: decoder = %q", p.Decoder)
	}
}

func TestGPUDecodeTreatsUnknownFormatAsTenBit(t *testing.T) {
	src := SourceVideo{Codec: "hevc", Width: 5760, Height: 2880}
	if p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, false), src); p.Decoder != "" {
		t.Errorf("unknown pixel format decoded on the GPU without scale_cuda: %q", p.Decoder)
	}
	known := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, true), SourceVideo{Codec: "hevc", Width: 5760, Height: 2880, PixFmt: "yuv420p"})
	unknown := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, true), src)
	if unknown.decodeVRAMMiB <= known.decodeVRAMMiB {
		t.Errorf("unknown format reserves %d MiB, 8-bit %d MiB; unknown should assume the larger 10-bit frames", unknown.decodeVRAMMiB, known.decodeVRAMMiB)
	}
}

func TestGPUDecodeSkipsUnsupportedSources(t *testing.T) {
	hw := gpuDecodeHardware(0, true)
	cases := []SourceVideo{
		{Codec: "hevc", Width: 3840, Height: 1920, PixFmt: "yuv422p10le"},
		{Codec: "mpeg4", Width: 1920, Height: 1080, PixFmt: "yuv420p"},
		{Codec: "hevc", PixFmt: "yuv420p"},
	}
	for _, src := range cases {
		if p := planGPUDecode(CodecHEVC, hw, src); p.Decoder != "" {
			t.Errorf("%+v decoded on the GPU with %q", src, p.Decoder)
		}
	}
	cpu := Hardware{Preferred: hw.Available[1], Available: hw.Available[1:]}
	if p := planGPUDecode(CodecX264, cpu, SourceVideo{Codec: "hevc", Width: 3840, Height: 1920, PixFmt: "yuv420p"}); p.Decoder != "" {
		t.Errorf("x264 plan decoded on the GPU with %q", p.Decoder)
	}
}

func TestGPUDecodeNeedsVRAMForDecoderAndEncoder(t *testing.T) {
	src := SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}
	encodeOnly := vramNeedMiB(CodecHEVC, 8192, 4096) + vramSafetyMiB
	decode := decodeVRAMNeedMiB(8192, 4096, 8192, 4096, false)

	tight := planGPUDecode(CodecHEVC, gpuDecodeHardware(encodeOnly+decode-1, false), src)
	if tight.Decoder != "" {
		t.Errorf("decoded on the GPU with %d MiB free, short of encoder and decoder", encodeOnly+decode-1)
	}
	if tight.Scale || tight.Unavailable {
		t.Errorf("a GPU decoder that does not fit must not shrink or refuse the stream: %dx%d %v", tight.Width, tight.Height, tight.Notes)
	}
	if !strings.Contains(strings.Join(tight.Notes, " "), "decoding on the CPU") {
		t.Errorf("notes should say why the CPU decodes: %v", tight.Notes)
	}

	roomy := planGPUDecode(CodecHEVC, gpuDecodeHardware(encodeOnly+decode, false), src)
	if roomy.Decoder != "hevc_cuvid" {
		t.Errorf("not decoded on the GPU with %d MiB free: %v", encodeOnly+decode, roomy.Notes)
	}
}

func TestReserveVRAMIncludesGPUDecode(t *testing.T) {
	src := SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}
	p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, false), src)
	release := p.ReserveVRAM()
	got := pendingVRAMByGPU()[1]
	release()
	want := vramNeedMiB(CodecHEVC, 8192, 4096) + decodeVRAMNeedMiB(8192, 4096, 8192, 4096, false)
	if got != want {
		t.Errorf("reserved %d MiB, want encoder and decoder %d MiB", got, want)
	}
	if left := pendingVRAMByGPU()[1]; left != 0 {
		t.Errorf("%d MiB still reserved after release", left)
	}
}

func TestSourceFormat(t *testing.T) {
	cases := []struct {
		pixFmt          string
		high, chroma420 bool
	}{
		{"yuv420p", false, true},
		{"yuvj420p", false, true},
		{"yuv420p10le", true, true},
		{"p010le", true, true},
		{"", true, true},
		{"yuv422p", false, false},
		{"yuv444p10le", false, false},
	}
	for _, c := range cases {
		high, chroma420 := sourceFormat(c.pixFmt)
		if high != c.high || chroma420 != c.chroma420 {
			t.Errorf("sourceFormat(%q) = %v, %v; want %v, %v", c.pixFmt, high, chroma420, c.high, c.chroma420)
		}
	}
}

func TestListsName(t *testing.T) {
	decoders := " V..... h264                 H.264 / AVC\n V..... hevc_cuvid           Nvidia CUVID HEVC decoder (codec hevc)\n"
	if !listsName(decoders, "hevc_cuvid", 1) {
		t.Error("hevc_cuvid not found in the decoder listing")
	}
	if listsName(decoders, "av1_cuvid", 1) {
		t.Error("av1_cuvid found although it is not listed")
	}
	if !listsName("Hardware acceleration methods:\ncuda\nvaapi\n", "cuda", 0) {
		t.Error("cuda not found in the hwaccel listing")
	}
	if !listsName(" ... scale_cuda        V->V       GPU accelerated video resizer\n", "scale_cuda", 1) {
		t.Error("scale_cuda not found in the filter listing")
	}
}

func TestGPUDecodeExplainsCPUDecoding(t *testing.T) {
	cases := map[string]SourceVideo{
		"could not be read": {Codec: "hevc", Width: 5760, Height: 2880},
		"no scale_cuda":     {Codec: "hevc", Width: 5760, Height: 2880, PixFmt: "yuv420p10le"},
		"only 4:2:0":        {Codec: "hevc", Width: 5760, Height: 2880, PixFmt: "yuv422p"},
	}
	for want, src := range cases {
		p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, false), src)
		if notes := strings.Join(p.Notes, " "); p.Decoder != "" || !strings.Contains(notes, want) {
			t.Errorf("%+v: decoder %q, notes %q; want CPU decoding explained with %q", src, p.Decoder, notes, want)
		}
	}
}

func TestHardwareMayDecodeOnGPU(t *testing.T) {
	hw := gpuDecodeHardware(0, false)
	if !hw.MayDecodeOnGPU("HEVC") || hw.MayDecodeOnGPU("av1") {
		t.Errorf("MayDecodeOnGPU: hevc %v, av1 %v; want true, false", hw.MayDecodeOnGPU("HEVC"), hw.MayDecodeOnGPU("av1"))
	}
	if (Hardware{}).MayDecodeOnGPU("hevc") {
		t.Error("no hardware should decode nothing on the GPU")
	}
}

// The free memory of a GPU shared with another program, with the measured peak of each encode
// while its output was blocked. The plan must not pick GPU decoding where that peak, plus the
// safety margin, does not fit.
func TestGPUDecodeEstimatesCoverMeasuredPeaks(t *testing.T) {
	cases := []struct {
		codec   string
		src     SourceVideo
		peakMiB int
	}{
		{CodecH264, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}, 2369},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 7200, Height: 3600, PixFmt: "yuv420p"}, 2115},
		{CodecH264, SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"}, 1849},
		{CodecHEVC, SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"}, 2138},
		{CodecHEVC, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"}, 1325},
		{CodecH264, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"}, 1485},
	}
	for _, c := range cases {
		p := planGPUDecode(c.codec, gpuDecodeHardware(0, false), c.src)
		if p.Decoder == "" {
			t.Fatalf("%s from %dx%d: no GPU decoder chosen", c.codec, c.src.Width, c.src.Height)
		}
		if est := vramNeedMiB(p.Codec, p.Width, p.Height) + p.decodeVRAMMiB; est < c.peakMiB {
			t.Errorf("%s from %dx%d to %dx%d: estimate %d MiB is below the measured %d MiB", c.codec, c.src.Width, c.src.Height, p.Width, p.Height, est, c.peakMiB)
		}
	}
	// Full-size 7-8K HEVC ran out of 2.5 GB with the output blocked.
	for _, src := range []SourceVideo{{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}, {Codec: "hevc", Width: 7200, Height: 3600, PixFmt: "yuv420p"}} {
		if p := planGPUDecode(CodecHEVC, gpuDecodeHardware(2581, false), src); p.Decoder != "" {
			t.Errorf("HEVC from %dx%d decoded on the GPU with 2581 MiB free", src.Width, src.Height)
		}
	}
}
