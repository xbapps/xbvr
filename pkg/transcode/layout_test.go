package transcode

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func testLayout(t *testing.T, duration float64) Layout {
	t.Helper()
	l, err := NewLayout(duration, Options{BitrateMbps: 20, AudioKbps: 128})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestNewLayoutRejectsNoDuration(t *testing.T) {
	if _, err := NewLayout(0, Options{}); err == nil {
		t.Fatal("a stream with no duration has no layout")
	}
}

func TestLayoutCoversDurationExactly(t *testing.T) {
	cases := []struct {
		duration float64
		chunks   int
	}{
		{1, 1},
		{2, 1},
		{2.001, 2},
		{2527.175, 1264},
	}
	for _, c := range cases {
		l := testLayout(t, c.duration)
		if l.Chunks != c.chunks {
			t.Errorf("duration %v: chunks = %d, want %d", c.duration, l.Chunks, c.chunks)
		}
		var total int64
		for _, entry := range parseSidxEntries(t, l) {
			total += entry.duration
			if entry.duration <= 0 {
				t.Errorf("duration %v: empty sidx entry", c.duration)
			}
		}
		if want := int64(c.duration*1000 + 0.5); total != want {
			t.Errorf("duration %v: sidx covers %d ms, want %d", c.duration, total, want)
		}
	}
}

func TestLayoutLimitsEntryCount(t *testing.T) {
	l := testLayout(t, 200000)
	if l.Chunks > maxChunks {
		t.Fatalf("chunks = %d exceeds the sidx limit", l.Chunks)
	}
	entries := parseSidxEntries(t, l)
	var total int64
	for _, entry := range entries {
		total += entry.duration
	}
	if total != 200000000 {
		t.Fatalf("sidx covers %d ms, want 200000000", total)
	}
}

func TestChunkAt(t *testing.T) {
	l := testLayout(t, 100)
	cases := []struct {
		offset int64
		chunk  int
		exact  bool
	}{
		{0, 0, false},
		{l.DataStart - 1, 0, false},
		{l.DataStart, 0, true},
		{l.DataStart + 1, 0, false},
		{l.ChunkOffset(7), 7, true},
		{l.ChunkOffset(7) + 3, 7, false},
		{l.ChunkOffset(l.Chunks-1) + l.ChunkSize, l.Chunks - 1, false},
		{l.Total - 1, l.Chunks - 1, false},
	}
	for _, c := range cases {
		chunk, exact := l.ChunkAt(c.offset)
		if chunk != c.chunk || exact != c.exact {
			t.Errorf("ChunkAt(%d) = (%d, %v), want (%d, %v)", c.offset, chunk, exact, c.chunk, c.exact)
		}
	}
	if got := l.StartSeconds(7); got != 14 {
		t.Errorf("StartSeconds(7) = %v, want 14", got)
	}
}

func TestHeaderIsExactlyDataStart(t *testing.T) {
	l := testLayout(t, 100)
	prefix := testBox("ftyp", []byte("iso5iso6mp41"))
	prefix = append(prefix, testMoov(1, 15360)...)
	header, err := l.Header(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(header)) != l.DataStart {
		t.Fatalf("header = %d bytes, want %d", len(header), l.DataStart)
	}
	if !bytes.HasPrefix(header, prefix) {
		t.Fatal("header must begin with the encoder's ftyp and moov")
	}
	// Every byte must belong to a top-level box, or a player would misread the stream.
	var names []string
	for rest := header; len(rest) > 0; {
		size := int(binary.BigEndian.Uint32(rest[:4]))
		if size < 8 || size > len(rest) {
			t.Fatalf("bad top-level box size %d", size)
		}
		names = append(names, string(rest[4:8]))
		rest = rest[size:]
	}
	want := []string{"ftyp", "moov", "sidx", "free"}
	if len(names) != len(want) {
		t.Fatalf("boxes = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("boxes = %v, want %v", names, want)
		}
	}
}

func TestHeaderOffsetsResolveToChunks(t *testing.T) {
	l := testLayout(t, 100)
	prefix := testBox("ftyp", []byte("iso5iso6mp41"))
	header, err := l.Header(prefix)
	if err != nil {
		t.Fatal(err)
	}

	// The same arithmetic a player uses: entries begin first_offset after the end of sidx.
	sidx := header[len(prefix):]
	sidxSize := int64(binary.BigEndian.Uint32(sidx[:4]))
	endOfSidx := int64(len(prefix)) + sidxSize
	offset := endOfSidx + int64(binary.BigEndian.Uint32(sidx[24:28]))
	if offset != l.DataStart {
		t.Fatalf("first chunk resolves to %d, want %d", offset, l.DataStart)
	}
	for i := 0; i < l.Chunks; i++ {
		if offset != l.ChunkOffset(i) {
			t.Fatalf("chunk %d resolves to %d, want %d", i, offset, l.ChunkOffset(i))
		}
		entry := sidx[sidxFixedSize+sidxEntrySize*i:]
		first := binary.BigEndian.Uint32(entry[:4])
		if first&0x80000000 != 0 {
			t.Fatalf("chunk %d is an indirect reference", i)
		}
		offset += int64(first)
	}
}

func TestHeaderRejectsOversizedPrefix(t *testing.T) {
	l := testLayout(t, 10)
	if _, err := l.Header(make([]byte, initReserve-freeHeaderSize+1)); err == nil {
		t.Fatal("an ftyp and moov that do not fit must be rejected")
	}
}

type sidxEntry struct {
	size     uint32
	duration int64
}

// parseSidxEntries reads the box the way a player does.
func parseSidxEntries(t *testing.T, l Layout) []sidxEntry {
	t.Helper()
	box := l.segmentIndex(8)
	if string(box[4:8]) != "sidx" || box[8] != 0 {
		t.Fatalf("not a version 0 sidx: %x", box[:12])
	}
	if got := binary.BigEndian.Uint32(box[16:20]); got != sidxTimescale {
		t.Fatalf("timescale = %d", got)
	}
	count := int(binary.BigEndian.Uint16(box[30:32]))
	if count != l.Chunks {
		t.Fatalf("reference_count = %d, want %d", count, l.Chunks)
	}
	if want := sidxFixedSize + sidxEntrySize*count; len(box) != want {
		t.Fatalf("sidx is %d bytes, want %d", len(box), want)
	}
	entries := make([]sidxEntry, count)
	for i := range entries {
		entry := box[sidxFixedSize+sidxEntrySize*i:]
		entries[i] = sidxEntry{
			size:     binary.BigEndian.Uint32(entry[:4]),
			duration: int64(binary.BigEndian.Uint32(entry[4:8])),
		}
	}
	return entries
}
