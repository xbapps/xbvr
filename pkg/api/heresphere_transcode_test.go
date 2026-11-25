package api

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/transcode"
)

func TestTranscodeStreamName(t *testing.T) {
	cases := []struct {
		prefix, codec string
		mbps          int
		software      bool
		want          string
	}{
		{"", transcode.CodecH264, 20, false, "H264 20 Mbps (hwe)"},
		{"", transcode.CodecHEVC, 10, false, "HEVC 10 Mbps (hwe)"},
		{"File 2/3", transcode.CodecAV1, 20, false, "File 2/3 AV1 20 Mbps (hwe)"},
		{"", transcode.CodecX264, 20, true, "x264 20 Mbps (swe)"},
		{"", transcode.CodecH264, 5, true, "H264 5 Mbps (swe)"},
	}
	for _, c := range cases {
		if got := transcodeStreamName(c.prefix, c.codec, c.mbps, c.software); got != c.want {
			t.Errorf("transcodeStreamName(%q, %q, %d, %v) = %q, want %q", c.prefix, c.codec, c.mbps, c.software, got, c.want)
		}
	}
}

func setTranscodeBitrates(t *testing.T, bitrates []int) {
	t.Helper()
	saved := config.Config.Interfaces.Heresphere.TranscodeBitrates
	config.Config.Interfaces.Heresphere.TranscodeBitrates = bitrates
	t.Cleanup(func() { config.Config.Interfaces.Heresphere.TranscodeBitrates = saved })
}

func TestTranscodeBitrateOnlyAcceptsOfferedValues(t *testing.T) {
	setTranscodeBitrates(t, []int{40, 10})
	cases := map[string]struct {
		mbps int
		ok   bool
	}{
		"":    {10, true},
		"40":  {40, true},
		" 10": {10, true},
		"20":  {0, false},
		"abc": {0, false},
		"-10": {0, false},
	}
	for param, want := range cases {
		mbps, ok := transcodeBitrate(param)
		if mbps != want.mbps || ok != want.ok {
			t.Errorf("transcodeBitrate(%q) = %d, %v; want %d, %v", param, mbps, ok, want.mbps, want.ok)
		}
	}
}

func TestTranscodeBitrateDefaultsWhenNoneConfigured(t *testing.T) {
	setTranscodeBitrates(t, nil)
	if got := transcodeBitrates(); len(got) != 1 || got[0] != transcode.DefaultBitrateMbps {
		t.Fatalf("transcodeBitrates() = %v, want [%d]", got, transcode.DefaultBitrateMbps)
	}
	if mbps, ok := transcodeBitrate(""); !ok || mbps != transcode.DefaultBitrateMbps {
		t.Fatalf("transcodeBitrate(\"\") = %d, %v", mbps, ok)
	}
}
