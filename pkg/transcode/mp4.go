package transcode

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// maxMP4Header bounds the prefix buffered while looking for the moov box. A fragmented MP4
// writer emits ftyp and moov before the first fragment, so this is far larger than needed.
const maxMP4Header = 8 << 20

// ReadMP4Header reads the ftyp and moov boxes from a fragmented MP4 stream and writes the
// known duration into every movie, track and media duration field. A fragmented MP4 normally
// carries zero in those fields, which makes players report a live stream with no length and
// disable seeking. The returned bytes must be written before the rest of r is copied.
func ReadMP4Header(r io.Reader, durationSeconds float64) ([]byte, error) {
	if durationSeconds <= 0 {
		return nil, errors.New("mp4: duration must be positive")
	}

	var prefix []byte
	for len(prefix) < maxMP4Header {
		size, typ, header, err := readBoxHeader(r)
		if err != nil {
			return prefix, err
		}
		if size < int64(len(header)) || size > maxMP4Header {
			return prefix, fmt.Errorf("mp4: invalid %q box size %d", typ, size)
		}
		body := make([]byte, size-int64(len(header)))
		if _, err := io.ReadFull(r, body); err != nil {
			return prefix, err
		}
		start := len(prefix)
		prefix = append(prefix, header...)
		prefix = append(prefix, body...)

		switch typ {
		case "moov":
			patchMoov(prefix[start:], durationSeconds)
			return prefix, nil
		case "ftyp", "free", "skip", "wide":
			// Boxes that can precede moov in a fragmented MP4.
		default:
			return prefix, fmt.Errorf("mp4: unexpected %q box before moov", typ)
		}
	}
	return prefix, errors.New("mp4: moov box not found")
}

// readBoxHeader reads an MP4 box header and returns the total box size, its type and the exact
// header bytes (including the 64-bit size when size == 1).
func readBoxHeader(r io.Reader) (int64, string, []byte, error) {
	head := make([]byte, 8)
	if _, err := io.ReadFull(r, head); err != nil {
		return 0, "", nil, err
	}
	size := int64(binary.BigEndian.Uint32(head[:4]))
	typ := string(head[4:])
	if size == 1 {
		ext := make([]byte, 8)
		if _, err := io.ReadFull(r, ext); err != nil {
			return 0, "", nil, err
		}
		size = int64(binary.BigEndian.Uint64(ext))
		head = append(head, ext...)
	}
	return size, typ, head, nil
}

// patchMoov walks a moov box and rewrites the duration fields it finds. It returns the movie
// timescale so tkhd, which counts in movie time, can be written too.
func patchMoov(moov []byte, seconds float64) uint32 {
	var movieTimescale uint32
	walkBoxes(moov[8:], func(typ string, payload []byte) {
		if typ == "mvhd" {
			if ts := patchTimeHeader(payload, seconds, 0); ts > 0 {
				movieTimescale = ts
			}
		}
	})
	// The first pass above fixes mvhd; tkhd needs the timescale from it, so walk again once
	// the value is known.
	walkBoxes(moov[8:], func(typ string, payload []byte) {
		switch typ {
		case "tkhd":
			patchTkhd(payload, seconds, movieTimescale)
		case "mdhd":
			patchTimeHeader(payload, seconds, 0)
		case "mehd":
			patchMehd(payload, seconds, movieTimescale)
		}
	})
	return movieTimescale
}

// walkBoxes calls fn for every box at this level and recurses into the containers that hold
// the headers we care about.
func walkBoxes(b []byte, fn func(typ string, payload []byte)) {
	for len(b) >= 8 {
		size := int64(binary.BigEndian.Uint32(b[:4]))
		typ := string(b[4:8])
		hdr := int64(8)
		if size == 1 {
			if len(b) < 16 {
				return
			}
			size = int64(binary.BigEndian.Uint64(b[8:16]))
			hdr = 16
		}
		if size < hdr || size > int64(len(b)) {
			return
		}
		payload := b[hdr:size]
		fn(typ, payload)
		switch typ {
		case "trak", "mdia", "minf", "stbl", "mvex", "edts", "dinf", "udta":
			walkBoxes(payload, fn)
		}
		b = b[size:]
	}
}

// patchTimeHeader writes seconds into an mvhd, tkhd or mdhd payload. The duration unit is the
// box's own timescale; fallback is used when the box has no timescale of its own (tkhd).
func patchTimeHeader(payload []byte, seconds float64, fallback uint32) uint32 {
	if len(payload) < 4 {
		return 0
	}
	version := payload[0]
	switch version {
	case 0:
		// creation, modification, timescale, duration
		if len(payload) < 20 {
			return 0
		}
		timescale := binary.BigEndian.Uint32(payload[12:16])
		if timescale == 0 {
			timescale = fallback
		}
		if timescale == 0 {
			return 0
		}
		binary.BigEndian.PutUint32(payload[16:20], durationUnits(seconds, timescale))
		return timescale
	case 1:
		// creation, modification, timescale, duration (64-bit)
		if len(payload) < 32 {
			return 0
		}
		timescale := binary.BigEndian.Uint32(payload[20:24])
		if timescale == 0 {
			timescale = fallback
		}
		if timescale == 0 {
			return 0
		}
		binary.BigEndian.PutUint64(payload[24:32], durationUnits64(seconds, timescale))
		return timescale
	}
	return 0
}

// patchTkhd writes the track duration. Unlike mvhd and mdhd, tkhd has track_ID and reserved
// fields before the duration and always counts in the movie timescale.
func patchTkhd(payload []byte, seconds float64, movieTimescale uint32) {
	if movieTimescale == 0 || len(payload) < 4 {
		return
	}
	switch payload[0] {
	case 0:
		if len(payload) >= 24 {
			binary.BigEndian.PutUint32(payload[20:24], durationUnits(seconds, movieTimescale))
		}
	case 1:
		if len(payload) >= 36 {
			binary.BigEndian.PutUint64(payload[28:36], durationUnits64(seconds, movieTimescale))
		}
	}
}

// patchMehd writes the overall fragment duration in the movie timescale.
func patchMehd(payload []byte, seconds float64, timescale uint32) {
	if timescale == 0 || len(payload) < 4 {
		return
	}
	version := payload[0]
	if version == 0 && len(payload) >= 8 {
		binary.BigEndian.PutUint32(payload[4:8], durationUnits(seconds, timescale))
	} else if version == 1 && len(payload) >= 12 {
		binary.BigEndian.PutUint64(payload[4:12], durationUnits64(seconds, timescale))
	}
}

// TrackTimescales maps the track IDs in a moov box to their media timescales. Fragment
// timestamps are written in each track's own timescale, so both values are needed when a
// stream is shifted to start at a seek position. The input may be the full ftyp and moov
// prefix returned by ReadMP4Header.
func TrackTimescales(header []byte) map[uint32]uint32 {
	timescales := map[uint32]uint32{}
	for len(header) >= 8 {
		size := int64(binary.BigEndian.Uint32(header[:4]))
		hdr := int64(8)
		if size == 1 {
			if len(header) < 16 {
				return timescales
			}
			size = int64(binary.BigEndian.Uint64(header[8:16]))
			hdr = 16
		}
		if size < hdr || size > int64(len(header)) {
			return timescales
		}
		if string(header[4:8]) == "moov" {
			walkBoxes(header[hdr:size], func(typ string, payload []byte) {
				if typ != "trak" {
					return
				}
				var trackID, timescale uint32
				walkBoxes(payload, func(typ string, payload []byte) {
					switch typ {
					case "tkhd":
						trackID = tkhdTrackID(payload)
					case "mdhd":
						timescale = mdhdTimescale(payload)
					}
				})
				if trackID > 0 && timescale > 0 {
					timescales[trackID] = timescale
				}
			})
			return timescales
		}
		header = header[size:]
	}
	return timescales
}

func tkhdTrackID(payload []byte) uint32 {
	if len(payload) < 4 {
		return 0
	}
	switch payload[0] {
	case 0:
		if len(payload) >= 16 {
			return binary.BigEndian.Uint32(payload[12:16])
		}
	case 1:
		if len(payload) >= 24 {
			return binary.BigEndian.Uint32(payload[20:24])
		}
	}
	return 0
}

func mdhdTimescale(payload []byte) uint32 {
	if len(payload) < 4 {
		return 0
	}
	switch payload[0] {
	case 0:
		if len(payload) >= 16 {
			return binary.BigEndian.Uint32(payload[12:16])
		}
	case 1:
		if len(payload) >= 24 {
			return binary.BigEndian.Uint32(payload[20:24])
		}
	}
	return 0
}

// maxMoofSize bounds the fragment header held in memory while timestamps are shifted.
const maxMoofSize = 64 << 20

// BoxObserver is told where top-level boxes begin and end in a stream being written, so that
// extra boxes can be inserted without splitting one.
type BoxObserver interface {
	BoxStart()
	BoxEnd()
}

// PatchFragmentTimes copies a fragmented MP4 stream and shifts every fragment's tfdt box by
// offsetSeconds. A fresh encode starts its timestamps at zero, so the shift places each encode
// at its real position in the source. moov is the header already returned by ReadMP4Header.
func PatchFragmentTimes(r io.Reader, w io.Writer, moov []byte, offsetSeconds float64) error {
	boxStart, boxEnd := func() {}, func() {}
	if observer, ok := w.(BoxObserver); ok {
		boxStart, boxEnd = observer.BoxStart, observer.BoxEnd
	}
	timescales := TrackTimescales(moov)
	for {
		size, typ, header, err := readBoxHeader(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		boxStart()
		if size == 0 {
			// A zero-sized box runs to the end of the stream.
			if _, err := w.Write(header); err != nil {
				return err
			}
			_, err := io.Copy(w, r)
			return err
		}
		headerLen := int64(len(header))
		if size < headerLen {
			return fmt.Errorf("mp4: invalid %q box size %d", typ, size)
		}
		bodyLen := size - headerLen
		if typ == "moof" {
			if bodyLen > maxMoofSize {
				return fmt.Errorf("mp4: %q box of %d bytes is too large", typ, bodyLen)
			}
			body := make([]byte, int(bodyLen))
			if _, err := io.ReadFull(r, body); err != nil {
				return err
			}
			patchMoofTimes(body, timescales, offsetSeconds)
			if _, err := w.Write(header); err != nil {
				return err
			}
			if _, err := w.Write(body); err != nil {
				return err
			}
			boxEnd()
			continue
		}
		if _, err := w.Write(header); err != nil {
			return err
		}
		if bodyLen > 0 {
			if _, err := io.CopyN(w, r, bodyLen); err != nil {
				return err
			}
		}
		boxEnd()
	}
}

func patchMoofTimes(moof []byte, timescales map[uint32]uint32, offsetSeconds float64) {
	if len(moof) < 8 {
		return
	}
	walkBoxes(moof, func(typ string, payload []byte) {
		if typ == "traf" {
			patchTrafTimes(payload, timescales, offsetSeconds)
		}
	})
}

func patchTrafTimes(traf []byte, timescales map[uint32]uint32, offsetSeconds float64) {
	var trackID uint32
	walkBoxes(traf, func(typ string, payload []byte) {
		if typ == "tfhd" {
			trackID = tfhdTrackID(payload)
		}
	})
	timescale, ok := timescales[trackID]
	if trackID == 0 || !ok || timescale == 0 {
		return
	}
	delta := uint64(math.Round(offsetSeconds * float64(timescale)))
	walkBoxes(traf, func(typ string, payload []byte) {
		if typ == "tfdt" {
			shiftTFDT(payload, delta)
		}
	})
}

func tfhdTrackID(payload []byte) uint32 {
	if len(payload) < 8 {
		return 0
	}
	flags := binary.BigEndian.Uint32(payload[:4]) & 0xffffff
	offset := 4
	if flags&0x01 != 0 {
		offset += 8 // base-data-offset is present
	}
	if len(payload) < offset+4 {
		return 0
	}
	return binary.BigEndian.Uint32(payload[offset : offset+4])
}

func shiftTFDT(payload []byte, delta uint64) {
	if len(payload) < 4 {
		return
	}
	switch payload[0] {
	case 0:
		if len(payload) >= 8 {
			value := uint64(binary.BigEndian.Uint32(payload[4:8]))
			value = addSaturated(value, delta, math.MaxUint32)
			binary.BigEndian.PutUint32(payload[4:8], uint32(value))
		}
	case 1:
		if len(payload) >= 12 {
			value := binary.BigEndian.Uint64(payload[4:12])
			value = addSaturated(value, delta, math.MaxUint64)
			binary.BigEndian.PutUint64(payload[4:12], value)
		}
	}
}

func addSaturated(value, delta, limit uint64) uint64 {
	if value >= limit || delta >= limit-value {
		return limit
	}
	return value + delta
}

func durationUnits(seconds float64, timescale uint32) uint32 {
	n := math.Round(seconds * float64(timescale))
	if n < 0 {
		return 0
	}
	if n > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(n)
}

func durationUnits64(seconds float64, timescale uint32) uint64 {
	n := math.Round(seconds * float64(timescale))
	if n < 0 {
		return 0
	}
	if n > math.MaxUint64 {
		return math.MaxUint64
	}
	return uint64(n)
}
