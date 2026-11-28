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
	if unknown.gpuDecodeVRAMMiB <= known.gpuDecodeVRAMMiB {
		t.Errorf("unknown format reserves %d MiB, 8-bit %d MiB; unknown should assume the larger 10-bit frames", unknown.gpuDecodeVRAMMiB, known.gpuDecodeVRAMMiB)
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
	total := gpuDecodeVRAMNeedMiB(CodecHEVC, 8192, 4096, 8192, 4096, false) + vramSafetyMiB

	tight := planGPUDecode(CodecHEVC, gpuDecodeHardware(total-1, false), src)
	if tight.Decoder != "" {
		t.Errorf("decoded on the GPU with %d MiB free, short of encoder and decoder", total-1)
	}
	if tight.Scale || tight.Unavailable {
		t.Errorf("a GPU decoder that does not fit must not shrink or refuse the stream: %dx%d %v", tight.Width, tight.Height, tight.Notes)
	}
	if !strings.Contains(strings.Join(tight.Notes, " "), "decoding on the CPU") {
		t.Errorf("notes should say why the CPU decodes: %v", tight.Notes)
	}

	roomy := planGPUDecode(CodecHEVC, gpuDecodeHardware(total, false), src)
	if roomy.Decoder != "hevc_cuvid" {
		t.Errorf("not decoded on the GPU with %d MiB free: %v", total, roomy.Notes)
	}
}

func TestReserveVRAMIncludesGPUDecode(t *testing.T) {
	src := SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}
	p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, false), src)
	release := p.ReserveVRAM()
	got := pendingVRAMByGPU()[1]
	release()
	want := gpuDecodeVRAMNeedMiB(CodecHEVC, 8192, 4096, 8192, 4096, false)
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

// Peaks of GPU-decoded encodes with the output blocked, as when a player pauses, measured with
// jellyfin-ffmpeg. The estimate must cover each.
func TestGPUDecodeEstimatesCoverMeasuredPeaks(t *testing.T) {
	cases := []struct {
		codec   string
		src     SourceVideo
		peakMiB int
	}{
		{CodecH264, SourceVideo{Codec: "h264", Width: 1920, Height: 1080, PixFmt: "yuv420p"}, 728},
		{CodecHEVC, SourceVideo{Codec: "h264", Width: 1920, Height: 1080, PixFmt: "yuv420p"}, 729},
		{CodecAV1, SourceVideo{Codec: "h264", Width: 1920, Height: 1080, PixFmt: "yuv420p"}, 782},
		{CodecH264, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"}, 1281},
		{CodecHEVC, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"}, 1109},
		{CodecAV1, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"}, 1289},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 4096, Height: 2048, PixFmt: "yuv420p"}, 1313},
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 4096, Height: 2048, PixFmt: "yuv420p"}, 1149},
		{CodecAV1, SourceVideo{Codec: "hevc", Width: 4096, Height: 2048, PixFmt: "yuv420p"}, 1317},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 5400, Height: 2700, PixFmt: "yuv420p"}, 1471},
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 5400, Height: 2700, PixFmt: "yuv420p"}, 1506},
		{CodecAV1, SourceVideo{Codec: "hevc", Width: 5400, Height: 2700, PixFmt: "yuv420p"}, 1917},
		{CodecH264, SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"}, 1573},
		{CodecHEVC, SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"}, 1682},
		{CodecAV1, SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"}, 2188},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 7200, Height: 3600, PixFmt: "yuv420p"}, 1731},
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 7200, Height: 3600, PixFmt: "yuv420p"}, 2216},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}, 1937},
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}, 2416},
		{CodecH264, SourceVideo{Codec: "av1", Width: 8192, Height: 4096, PixFmt: "yuv420p"}, 1863},
		{CodecHEVC, SourceVideo{Codec: "av1", Width: 8192, Height: 4096, PixFmt: "yuv420p"}, 2342},
	}
	hw := gpuDecodeHardware(0, true)
	hw.Preferred.Decoders = map[string]string{"h264": "h264_cuvid", "hevc": "hevc_cuvid", "av1": "av1_cuvid"}
	for _, c := range cases {
		p := planGPUDecode(c.codec, hw, c.src)
		if p.Decoder == "" {
			t.Fatalf("%s from %s %dx%d: no GPU decoder chosen", c.codec, c.src.Codec, c.src.Width, c.src.Height)
		}
		if est := p.gpuDecodeVRAMMiB; est < c.peakMiB {
			t.Errorf("%s from %s %dx%d to %dx%d: estimate %d MiB is below the measured %d MiB", c.codec, c.src.Codec, c.src.Width, c.src.Height, p.Width, p.Height, est, c.peakMiB)
		}
	}
}

// With 2581 MiB free beside another program, these encodes ran out of memory when the GPU
// decoded, or only just fitted, so they must decode on the CPU there. 8K to H.264 fitted with
// room to spare and should use the GPU.
func TestGPUDecodeOnASharedGPU(t *testing.T) {
	hw := gpuDecodeHardware(2581, true)
	cpu := []struct {
		codec string
		src   SourceVideo
	}{
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 7680, Height: 3840, PixFmt: "yuv420p10le"}},
		{CodecH264, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p10le"}},
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 7680, Height: 3840, PixFmt: "yuv420p10le"}},
	}
	for _, c := range cpu {
		if p := planGPUDecode(c.codec, hw, c.src); p.Decoder != "" {
			t.Errorf("%s from %dx%d %s decoded on the GPU with 2581 MiB free", c.codec, c.src.Width, c.src.Height, c.src.PixFmt)
		}
	}
	if p := planGPUDecode(CodecH264, hw, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}); p.Decoder == "" {
		t.Errorf("8K HEVC to H.264 should decode on the GPU with 2581 MiB free: %v", p.Notes)
	}
	// AV1 kept at full size above 7K ran out of memory with the GPU decoding.
	for _, size := range [][2]int{{7200, 3600}, {8192, 4096}} {
		if need := gpuDecodeVRAMNeedMiB(CodecAV1, size[0], size[1], size[0], size[1], false); need+vramSafetyMiB <= 2581 {
			t.Errorf("full-size AV1 from %dx%d estimated at %d MiB, which would fit in 2581 MiB", size[0], size[1], need)
		}
	}
}

func qsvDecodeHardware() Hardware {
	b := Backend{
		Name:         BackendQSV,
		Label:        "Intel Quick Sync",
		Encoders:     map[string]string{CodecH264: "h264_qsv", CodecHEVC: "hevc_qsv"},
		Device:       "/dev/dri/renderD128",
		Decoders:     map[string]string{"h264": "h264_qsv", "hevc": "hevc_qsv", "av1": "av1_qsv"},
		ConvertOnGPU: true,
	}
	cpu := Backend{Name: BackendCPU, Label: "CPU (x264)", Encoders: map[string]string{CodecH264: "libx264", CodecX264: "libx264"}}
	return Hardware{Preferred: b, Available: []Backend{b, cpu}, InitialBurst: true}
}

func TestQSVDecodeScalesInTheVideoProcessor(t *testing.T) {
	p := planGPUDecode(CodecH264, qsvDecodeHardware(), SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"})
	if p.Decoder != "hevc_qsv" {
		t.Fatalf("decoder = %q, want hevc_qsv: %v", p.Decoder, p.Notes)
	}
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 30, Options{}, p, -1), " ")
	want := "-init_hw_device qsv=hw:/dev/dri/renderD128 -filter_hw_device hw -xerror -hwaccel qsv -hwaccel_device hw -hwaccel_output_format qsv -c:v hevc_qsv"
	if !strings.Contains(args, want) || strings.Index(args, want) > strings.Index(args, "-i /videos") {
		t.Errorf("args should select the Quick Sync decoder on the encoder's device before the input: %s", args)
	}
	if !strings.Contains(args, "-vf vpp_qsv=w=4096:h=2048 ") {
		t.Errorf("args should scale with vpp_qsv: %s", args)
	}
	for _, unwanted := range []string{"hwupload", "scale=", "-resize", "-pix_fmt", "format=nv12"} {
		if strings.Contains(args, unwanted) {
			t.Errorf("8-bit Quick Sync frames need no %s: %s", unwanted, args)
		}
	}
}

func TestQSVDecodeFilters(t *testing.T) {
	hw := qsvDecodeHardware()
	cases := []struct {
		codec string
		src   SourceVideo
		want  string
	}{
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 7200, Height: 3600, PixFmt: "yuv420p"}, ""},
		{CodecHEVC, SourceVideo{Codec: "hevc", Width: 7200, Height: 3600, PixFmt: "yuv420p10le"}, "vpp_qsv=format=nv12"},
		{CodecH264, SourceVideo{Codec: "av1", Width: 8000, Height: 4000, PixFmt: "yuv420p10le"}, "vpp_qsv=w=4096:h=2048:format=nv12"},
		{CodecH264, SourceVideo{Codec: "h264", Width: 3840, Height: 1920}, "vpp_qsv=format=nv12"},
	}
	for _, c := range cases {
		p := planGPUDecode(c.codec, hw, c.src)
		if p.Decoder == "" {
			t.Errorf("%+v: not decoded on the GPU: %v", c.src, p.Notes)
			continue
		}
		if got := videoFilter(p); got != c.want {
			t.Errorf("%+v to %s: filter %q, want %q", c.src, c.codec, got, c.want)
		}
	}
}

func TestQSVCPUDecodeUploadsFrames(t *testing.T) {
	p := planGPUDecode(CodecH264, qsvDecodeHardware(), SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p"}).CPUDecode()
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, p, -1), " ")
	if strings.Contains(args, "-hwaccel") || strings.Contains(args, "vpp_qsv") || strings.Contains(args, "-xerror") {
		t.Errorf("CPU decode still selects the Quick Sync decoder: %s", args)
	}
	if !strings.Contains(args, "-vf scale=4096:2048,format=nv12,hwupload=extra_hw_frames=64") {
		t.Errorf("CPU decode should scale and upload the frames: %s", args)
	}
}

func TestQSVDecodeViaVAAPIDevice(t *testing.T) {
	hw := qsvDecodeHardware()
	hw.Preferred.qsvViaVAAPI = true
	hw.Available[0] = hw.Preferred
	p := planGPUDecode(CodecHEVC, hw, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"})
	args := strings.Join(Args("ffmpeg", "/videos/scene.mp4", 0, Options{}, p, -1), " ")
	want := "-init_hw_device vaapi=va:/dev/dri/renderD128 -init_hw_device qsv=hw@va -filter_hw_device hw -xerror -hwaccel qsv -hwaccel_device hw"
	if !strings.Contains(args, want) {
		t.Errorf("args should decode on the QSV device derived from VAAPI: %s", args)
	}
}

func TestQSVDecodeNeedsNoVRAM(t *testing.T) {
	hw := qsvDecodeHardware()
	if !hw.MayDecodeOnGPU("hevc") || hw.MayDecodeOnGPU("vp9") {
		t.Errorf("MayDecodeOnGPU: hevc %v, vp9 %v; want true, false", hw.MayDecodeOnGPU("hevc"), hw.MayDecodeOnGPU("vp9"))
	}
	p := planGPUDecode(CodecHEVC, hw, SourceVideo{Codec: "hevc", Width: 8192, Height: 4096, PixFmt: "yuv420p10le"})
	if p.Decoder != "hevc_qsv" || p.gpuDecodeVRAMMiB != 0 {
		t.Fatalf("decoder %q, VRAM %d MiB; want hevc_qsv and no VRAM: %v", p.Decoder, p.gpuDecodeVRAMMiB, p.Notes)
	}
	release := p.ReserveVRAM()
	defer release()
	if pending := pendingVRAMByGPU(); len(pending) != 0 {
		t.Errorf("Quick Sync reserved VRAM: %v", pending)
	}
	if p := planGPUDecode(CodecHEVC, hw, SourceVideo{Codec: "hevc", Width: 3840, Height: 1920, PixFmt: "yuv422p10le"}); p.Decoder != "" {
		t.Errorf("4:2:2 source decoded on the GPU with %q", p.Decoder)
	}
}

func TestQSVDecodesLargeH264OnTheCPU(t *testing.T) {
	hw := qsvDecodeHardware()
	p := planGPUDecode(CodecH264, hw, SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"})
	if p.Decoder != "" || !strings.Contains(strings.Join(p.Notes, " "), "4096x2304") {
		t.Errorf("5760x2880 H.264 on Quick Sync: decoder %q, notes %v; want the CPU and the limit explained", p.Decoder, p.Notes)
	}
	if p := planGPUDecode(CodecH264, hw, SourceVideo{Codec: "h264", Width: 3840, Height: 1920, PixFmt: "yuv420p"}); p.Decoder != "h264_qsv" {
		t.Errorf("3840x1920 H.264 not decoded on Quick Sync: %v", p.Notes)
	}
	if p := planGPUDecode(CodecHEVC, gpuDecodeHardware(0, true), SourceVideo{Codec: "h264", Width: 5760, Height: 2880, PixFmt: "yuv420p"}); p.Decoder != "h264_cuvid" {
		t.Errorf("the Quick Sync H.264 limit should not apply to NVDEC: %v", p.Notes)
	}
}
