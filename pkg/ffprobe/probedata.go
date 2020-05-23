package ffprobe

import (
	"time"
)

// StreamType represents a media stream type like video, audio, subtitles, etc
type StreamType string

const (
	// StreamAny means any type of stream
	StreamAny StreamType = ""
	// StreamVideo is a video stream
	StreamVideo StreamType = "video"
	// StreamAudio is an audio stream
	StreamAudio StreamType = "audio"
	// StreamSubtitle is a subtitle stream
	StreamSubtitle StreamType = "subtitle"
)

// TODO: FIXME: We should remove the ALL_CAPS variants some time in the future (golint hates them)
const (
	// STREAM_ANY deprecated, use StreamAny
	STREAM_ANY = StreamAny
	// STREAM_VIDEO deprecated, use StreamVideo
	STREAM_VIDEO = StreamVideo
	// STREAM_AUDIO deprecated, use StreamAudio
	STREAM_AUDIO = StreamAudio
	// STREAM_SUBTITLE deprecated, use StreamSubtitle
	STREAM_SUBTITLE = StreamSubtitle
)

// ProbeData is the root json data structure returned by an ffprobe.
type ProbeData struct {
	Streams []*Stream `json:"streams"`
	Format  *Format   `json:"format"`
}

// Format is a json data structure to represent formats
type Format struct {
	Filename         string      `json:"filename"`
	NBStreams        int         `json:"nb_streams"`
	NBPrograms       int         `json:"nb_programs"`
	FormatName       string      `json:"format_name"`
	FormatLongName   string      `json:"format_long_name"`
	StartTimeSeconds float64     `json:"start_time,string"`
	DurationSeconds  float64     `json:"duration,string"`
	Size             string      `json:"size"`
	BitRate          string      `json:"bit_rate"`
	ProbeScore       int         `json:"probe_score"`
	Tags             *FormatTags `json:"tags"`
}

// FormatTags is a json data structure to represent format tags
type FormatTags struct {
	MajorBrand       string `json:"major_brand"`
	MinorVersion     string `json:"minor_version"`
	CompatibleBrands string `json:"compatible_brands"`
	CreationTime     string `json:"creation_time"`
}

// Stream is a json data structure to represent streams.
// A stream can be a video, audio, subtitle, etc type of stream.
type Stream struct {
	Index              int               `json:"index"`
	CodecName          string            `json:"codec_name"`
	CodecLongName      string            `json:"codec_long_name"`
	CodecType          string            `json:"codec_type"`
	CodecTimeBase      string            `json:"codec_time_base"`
	CodecTagString     string            `json:"codec_tag_string"`
	CodecTag           string            `json:"codec_tag"`
	RFrameRate         string            `json:"r_frame_rate"`
	AvgFrameRate       string            `json:"avg_frame_rate"`
	TimeBase           string            `json:"time_base"`
	StartPts           int               `json:"start_pts"`
	StartTime          string            `json:"start_time"`
	DurationTs         uint64            `json:"duration_ts"`
	Duration           string            `json:"duration"`
	BitRate            string            `json:"bit_rate"`
	BitsPerRawSample   string            `json:"bits_per_raw_sample"`
	NbFrames           string            `json:"nb_frames"`
	Disposition        StreamDisposition `json:"disposition,omitempty"`
	Tags               StreamTags        `json:"tags,omitempty"`
	Profile            string            `json:"profile,omitempty"`
	Width              int               `json:"width"`
	Height             int               `json:"height"`
	HasBFrames         int               `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string            `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string            `json:"display_aspect_ratio,omitempty"`
	PixFmt             string            `json:"pix_fmt,omitempty"`
	Level              int               `json:"level,omitempty"`
	ColorRange         string            `json:"color_range,omitempty"`
	ColorSpace         string            `json:"color_space,omitempty"`
	SampleFmt          string            `json:"sample_fmt,omitempty"`
	SampleRate         string            `json:"sample_rate,omitempty"`
	Channels           int               `json:"channels,omitempty"`
	ChannelLayout      string            `json:"channel_layout,omitempty"`
	BitsPerSample      int               `json:"bits_per_sample,omitempty"`
}

// StreamDisposition is a json data structure to represent stream dispositions
type StreamDisposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
}

// StreamTags is a json data structure to represent stream tags
type StreamTags struct {
	Rotate       int    `json:"rotate,string,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	Language     string `json:"language,omitempty"`
	Title        string `json:"title,omitempty"`
	Encoder      string `json:"encoder,omitempty"`
	Location     string `json:"location,omitempty"`
}

// StartTime returns the start time of the media file as a time.Duration
func (f *Format) StartTime() (duration time.Duration) {
	return time.Duration(f.StartTimeSeconds * float64(time.Second))
}

// Duration returns the duration of the media file as a time.Duration
func (f *Format) Duration() (duration time.Duration) {
	return time.Duration(f.DurationSeconds * float64(time.Second))
}

// GetStreams returns all streams which are of the given type
func (p *ProbeData) GetStreams(streamType StreamType) (streams []Stream) {
	for _, s := range p.Streams {
		if s == nil {
			continue
		}
		switch streamType {
		case StreamAny:
			streams = append(streams, *s)
		default:
			if s.CodecType == string(streamType) {
				streams = append(streams, *s)
			}
		}
	}
	return streams
}

// GetFirstVideoStream returns the first video stream found
func (p *ProbeData) GetFirstVideoStream() (str *Stream) {
	for _, s := range p.Streams {
		if s == nil {
			continue
		}
		if s.CodecType == string(StreamVideo) {
			return s
		}
	}
	return nil
}

// GetFirstAudioStream returns the first audio stream found
func (p *ProbeData) GetFirstAudioStream() (str *Stream) {
	for _, s := range p.Streams {
		if s == nil {
			continue
		}
		if s.CodecType == string(StreamAudio) {
			return s
		}
	}
	return nil
}

// GetFirstSubtitleStream returns the first subtitle stream found
func (p *ProbeData) GetFirstSubtitleStream() (str *Stream) {
	for _, s := range p.Streams {
		if s == nil {
			continue
		}
		if s.CodecType == string(StreamSubtitle) {
			return s
		}
	}
	return nil
}
