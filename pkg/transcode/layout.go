package transcode

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	// ChunkSeconds is the spacing of seek points and the keyframe interval of every encode.
	ChunkSeconds = 2

	// maxChunks is the largest entry count a sidx box can hold.
	maxChunks = 65535

	// maxChunkSize is the largest referenced size a sidx entry can hold.
	maxChunkSize = math.MaxInt32

	// initReserve is the space kept for ftyp and moov. Reserving a fixed amount means every
	// chunk position is known before an encode has produced its header.
	initReserve = 16 << 10

	sidxTimescale   = 1000
	sidxFixedSize   = 32
	sidxEntrySize   = 12
	freeHeaderSize  = 8
	sidxSAPStart    = 0x90000000 // starts_with_SAP = 1, SAP_type = 1
	overshootFactor = 2
)

// Layout assigns byte positions to a live encode, which has no file behind it. The stream is
// a fixed-size header followed by chunks of equal nominal size, so a byte offset identifies a
// chunk, and therefore a time, without any encode having run.
//
// Only the offset of a chunk carries meaning. Real encoded chunks are larger or smaller than
// ChunkSize, and a response simply continues past the next nominal boundary.
type Layout struct {
	Chunks      int   // number of seek points
	ChunkMillis int64 // duration of every chunk except the last
	ChunkSize   int64 // nominal bytes per chunk
	DataStart   int64 // offset of the first chunk
	Total       int64 // size reported to the client, with room for encoder overshoot

	durationMillis int64
}

// NewLayout maps a source duration and the configured bitrates onto a Layout.
func NewLayout(durationSeconds float64, opts Options) (Layout, error) {
	durationMillis := int64(math.Round(durationSeconds * 1000))
	if durationMillis <= 0 {
		return Layout{}, errors.New("transcode: duration must be positive")
	}

	chunkMillis := int64(ChunkSeconds * 1000)
	if durationMillis > chunkMillis*maxChunks {
		chunkMillis = (durationMillis + maxChunks - 1) / maxChunks
	}
	chunks := (durationMillis + chunkMillis - 1) / chunkMillis

	chunkSize := opts.VirtualSize(float64(chunkMillis) / 1000)
	if chunkSize < 1 {
		chunkSize = 1
	}
	if chunkSize > maxChunkSize {
		chunkSize = maxChunkSize
	}

	dataStart := int64(initReserve + sidxFixedSize + sidxEntrySize*chunks)
	return Layout{
		Chunks:         int(chunks),
		ChunkMillis:    chunkMillis,
		ChunkSize:      chunkSize,
		DataStart:      dataStart,
		Total:          dataStart + chunks*chunkSize*overshootFactor,
		durationMillis: durationMillis,
	}, nil
}

// StartSeconds returns the source time at which a chunk begins.
func (l Layout) StartSeconds(chunk int) float64 {
	return float64(int64(chunk)*l.ChunkMillis) / 1000
}

// ChunkOffset returns the byte offset at which a chunk begins.
func (l Layout) ChunkOffset(chunk int) int64 {
	return l.DataStart + int64(chunk)*l.ChunkSize
}

// ChunkAt returns the chunk containing offset. exact is true only when offset is the start of
// that chunk, which is the only position a seek through the index can produce. An offset past
// the last chunk resolves to the last chunk and is never exact.
func (l Layout) ChunkAt(offset int64) (chunk int, exact bool) {
	if offset < l.DataStart {
		return 0, false
	}
	distance := offset - l.DataStart
	index := distance / l.ChunkSize
	if index >= int64(l.Chunks) {
		return l.Chunks - 1, false
	}
	return int(index), distance%l.ChunkSize == 0
}

// Header assembles the bytes before the first chunk: the encoder's ftyp and moov, the segment
// index, and padding up to DataStart. The result is always exactly DataStart bytes long.
func (l Layout) Header(prefix []byte) ([]byte, error) {
	padding := initReserve - len(prefix)
	if padding < freeHeaderSize {
		return nil, fmt.Errorf("transcode: %d-byte mp4 header does not fit the %d bytes reserved", len(prefix), initReserve)
	}
	header := make([]byte, 0, l.DataStart)
	header = append(header, prefix...)
	header = append(header, l.segmentIndex(uint32(padding))...)
	header = append(header, freeBox(padding)...)
	return header, nil
}

// segmentIndex builds a version 0 sidx box covering the whole duration. firstOffset is the
// distance from the end of this box to the first chunk.
func (l Layout) segmentIndex(firstOffset uint32) []byte {
	size := sidxFixedSize + sidxEntrySize*l.Chunks
	box := make([]byte, size)
	binary.BigEndian.PutUint32(box[0:4], uint32(size))
	copy(box[4:8], "sidx")
	// version and flags are zero
	binary.BigEndian.PutUint32(box[12:16], 1) // reference_ID
	binary.BigEndian.PutUint32(box[16:20], sidxTimescale)
	// earliest_presentation_time is zero
	binary.BigEndian.PutUint32(box[24:28], firstOffset)
	// reserved is zero
	binary.BigEndian.PutUint16(box[30:32], uint16(l.Chunks))

	for i := 0; i < l.Chunks; i++ {
		entry := box[sidxFixedSize+sidxEntrySize*i:]
		duration := l.ChunkMillis
		if i == l.Chunks-1 {
			duration = l.durationMillis - int64(i)*l.ChunkMillis
		}
		binary.BigEndian.PutUint32(entry[0:4], uint32(l.ChunkSize)) // reference_type 0
		binary.BigEndian.PutUint32(entry[4:8], uint32(duration))
		binary.BigEndian.PutUint32(entry[8:12], sidxSAPStart)
	}
	return box
}

// freeBox returns a top-level box of size bytes that players skip.
func freeBox(size int) []byte {
	box := make([]byte, size)
	binary.BigEndian.PutUint32(box[0:4], uint32(size))
	copy(box[4:8], "free")
	return box
}
