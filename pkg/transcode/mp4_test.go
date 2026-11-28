package transcode

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func testBox(typ string, payload ...[]byte) []byte {
	body := bytes.Join(payload, nil)
	size := uint32(8 + len(body))
	b := make([]byte, 8, size)
	binary.BigEndian.PutUint32(b[:4], size)
	copy(b[4:8], typ)
	return append(b, body...)
}

func testFullBox(typ string, version byte, payload []byte) []byte {
	return testBox(typ, append([]byte{version, 0, 0, 0}, payload...))
}

func testDurationBox(typ string, version byte, timescale uint32, durationOffset int) []byte {
	payload := make([]byte, durationOffset+4)
	payload[0] = version
	if version == 0 {
		binary.BigEndian.PutUint32(payload[12:16], timescale)
	}
	return testBox(typ, payload)
}

func durationField(t *testing.T, b []byte, typ string, offset int) uint32 {
	t.Helper()
	for len(b) >= 8 {
		size := int(binary.BigEndian.Uint32(b[:4]))
		name := string(b[4:8])
		if size < 8 || size > len(b) {
			t.Fatalf("bad box %q size %d", name, size)
		}
		if name == typ {
			if len(b) < offset+4 {
				t.Fatalf("%s too short", typ)
			}
			return binary.BigEndian.Uint32(b[offset : offset+4])
		}
		switch name {
		case "moov", "trak", "mdia", "mvex":
			if v := findDurationField(t, b[8:size], typ, offset); v != 0 {
				return v
			}
		}
		b = b[size:]
	}
	return 0
}

func findDurationField(t *testing.T, b []byte, typ string, offset int) uint32 {
	t.Helper()
	for len(b) >= 8 {
		size := int(binary.BigEndian.Uint32(b[:4]))
		name := string(b[4:8])
		if size < 8 || size > len(b) {
			t.Fatalf("bad box %q size %d", name, size)
		}
		if name == typ {
			return binary.BigEndian.Uint32(b[offset : offset+4])
		}
		switch name {
		case "trak", "mdia", "mvex":
			if v := findDurationField(t, b[8:size], typ, offset); v != 0 {
				return v
			}
		}
		b = b[size:]
	}
	return 0
}

func TestReadMP4HeaderSetsDurations(t *testing.T) {
	// mvhd: version/flags, creation, modification, timescale, duration
	mvhd := testFullBox("mvhd", 0, func() []byte {
		p := make([]byte, 16)
		binary.BigEndian.PutUint32(p[8:12], 1000)
		return p
	}())
	// tkhd: version/flags, creation, modification, track_ID, reserved, duration
	tkhd := testFullBox("tkhd", 0, func() []byte {
		p := make([]byte, 20)
		binary.BigEndian.PutUint32(p[8:12], 1)
		return p
	}())
	// mdhd: version/flags, creation, modification, timescale, duration
	mdhd := testFullBox("mdhd", 0, func() []byte {
		p := make([]byte, 16)
		binary.BigEndian.PutUint32(p[8:12], 15360)
		return p
	}())
	mehd := testFullBox("mehd", 0, make([]byte, 4))
	moov := testBox("moov", mvhd, testBox("trak", tkhd, testBox("mdia", mdhd)), testBox("mvex", mehd))
	stream := testBox("ftyp", []byte("iso5iso6mp41")) // 16 bytes

	header, err := ReadMP4Header(bytes.NewReader(append(stream, moov...)), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(header) != len(stream)+len(moov) {
		t.Fatalf("header length = %d, want %d", len(header), len(stream)+len(moov))
	}
	if got := durationField(t, header, "mvhd", 8+4+12); got != 10_000 {
		t.Errorf("mvhd duration = %d, want 10000", got)
	}
	if got := durationField(t, header, "tkhd", 8+4+16); got != 10_000 {
		t.Errorf("tkhd duration = %d, want 10000", got)
	}
	if got := durationField(t, header, "mdhd", 8+4+12); got != 153_600 {
		t.Errorf("mdhd duration = %d, want 153600", got)
	}
	if got := durationField(t, header, "mehd", 8+4); got != 10_000 {
		t.Errorf("mehd duration = %d, want 10000", got)
	}
}

func TestReadMP4HeaderRejectsMissingMoov(t *testing.T) {
	stream := testBox("ftyp", []byte("iso5iso6mp41"))
	stream = append(stream, testBox("mdat", make([]byte, 12))...)
	if _, err := ReadMP4Header(bytes.NewReader(stream), 10); err == nil {
		t.Fatal("a stream without moov should be rejected")
	}
}

func testMoov(trackID, timescale uint32) []byte {
	tkhd := testFullBox("tkhd", 0, func() []byte {
		p := make([]byte, 20)
		binary.BigEndian.PutUint32(p[8:12], trackID)
		return p
	}())
	mdhd := testFullBox("mdhd", 0, func() []byte {
		p := make([]byte, 16)
		binary.BigEndian.PutUint32(p[8:12], timescale)
		return p
	}())
	return testBox("moov", testBox("trak", tkhd, testBox("mdia", mdhd)))
}

func testMoof(trackID, baseDecodeTime uint32) []byte {
	tfhd := testFullBox("tfhd", 0, func() []byte {
		p := make([]byte, 8)
		binary.BigEndian.PutUint32(p[0:4], trackID)
		return p
	}())
	tfdt := testFullBox("tfdt", 0, func() []byte {
		p := make([]byte, 4)
		binary.BigEndian.PutUint32(p[0:4], baseDecodeTime)
		return p
	}())
	return testBox("moof", testFullBox("mfhd", 0, make([]byte, 4)),
		testBox("traf", tfhd, tfdt))
}

func tfdtField(t *testing.T, stream []byte) uint32 {
	t.Helper()
	i := bytes.Index(stream, []byte("tfdt"))
	if i < 0 || i+12 > len(stream) {
		t.Fatal("tfdt box not found")
	}
	return binary.BigEndian.Uint32(stream[i+8 : i+12])
}

func TestTrackTimescales(t *testing.T) {
	moov := testMoov(3, 15360)
	got := TrackTimescales(moov)
	if got[3] != 15360 {
		t.Errorf("TrackTimescales = %v, want track 3 at 15360", got)
	}
	withFtyp := append(testBox("ftyp", []byte("iso5iso6mp41")), moov...)
	if got := TrackTimescales(withFtyp); got[3] != 15360 {
		t.Errorf("TrackTimescales after ftyp = %v, want track 3 at 15360", got)
	}
}

func TestPatchFragmentTimesOffsetsTFDT(t *testing.T) {
	moov := testMoov(1, 1000)
	moof := testMoof(1, 0)
	stream := append(append([]byte{}, moov...), moof...)
	var out bytes.Buffer
	if err := PatchFragmentTimes(bytes.NewReader(stream), &out, moov, 12.5); err != nil {
		t.Fatal(err)
	}
	if got := tfdtField(t, out.Bytes()); got != 12500 {
		t.Errorf("tfdt = %d, want 12500", got)
	}
}

func TestPatchFragmentTimesLeavesDataBeforeMoof(t *testing.T) {
	moov := testMoov(1, 1000)
	moof := testMoof(1, 0)
	stream := append(append([]byte{}, moov...), moof...)
	stream = append(stream, testBox("mdat", []byte("video data"))...)
	var out bytes.Buffer
	if err := PatchFragmentTimes(bytes.NewReader(stream), &out, moov, 5); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("video data")) {
		t.Error("mdat payload was not copied")
	}
	if !bytes.Contains(out.Bytes(), []byte("moof")) {
		t.Error("moof box was not copied")
	}
}
