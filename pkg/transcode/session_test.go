package transcode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeEncoders records every encode the Manager starts and lets a test feed each one.
type fakeEncoders struct {
	mu       sync.Mutex
	seeks    []float64
	running  []*fakeEncoder
	prepared int
	prefix   []byte
	fail     error
}

type fakeEncoder struct {
	output chan []byte
	done   chan struct{}
	killed chan struct{}
}

// emit queues encoder output, which is delivered strictly in order after the encoder's header.
func (e *fakeEncoder) emit(b []byte) {
	e.output <- b
}

func testPrefix() []byte {
	return append(testBox("ftyp", []byte("iso5iso6mp41")), testMoov(1, 1000)...)
}

func (f *fakeEncoders) spawn(ctx context.Context, seek float64) (*Process, error) {
	if f.fail != nil {
		return nil, f.fail
	}
	reader, writer := io.Pipe()
	e := &fakeEncoder{output: make(chan []byte, 16), done: make(chan struct{}), killed: make(chan struct{})}
	f.mu.Lock()
	f.seeks = append(f.seeks, seek)
	f.running = append(f.running, e)
	prefix := f.prefix
	f.mu.Unlock()

	e.output <- prefix
	go func() {
		defer writer.Close()
		for {
			select {
			case <-ctx.Done():
				close(e.killed)
				return
			case b := <-e.output:
				if _, err := writer.Write(b); err != nil {
					<-ctx.Done()
					close(e.killed)
					return
				}
			}
		}
	}()
	return &Process{
		Stdout: reader,
		Wait: func() error {
			close(e.done)
			return nil
		},
	}, nil
}

func (f *fakeEncoders) prepare() (Spawner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prepared++
	return f.spawn, nil
}

func (f *fakeEncoders) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seeks)
}

func (f *fakeEncoders) last() *fakeEncoder {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running[len(f.running)-1]
}

// patchedPrefix is the fake encoder's header as it is sent, with the full duration written in.
func patchedPrefix(t *testing.T, f *fakeEncoders, src Source) []byte {
	t.Helper()
	prefix, err := ReadMP4Header(bytes.NewReader(f.prefix), src.Duration)
	if err != nil {
		t.Fatal(err)
	}
	return prefix
}

func newTestSource(t *testing.T, f *fakeEncoders) Source {
	t.Helper()
	l, err := NewLayout(100, Options{BitrateMbps: 1, AudioKbps: 128})
	if err != nil {
		t.Fatal(err)
	}
	f.prefix = testPrefix()
	return Source{Key: "file", Duration: 100, Layout: l, Prepare: f.prepare}
}

// reader runs a Playback and exposes what it has received so far.
type reader struct {
	mu     sync.Mutex
	got    bytes.Buffer
	result chan error
	cancel context.CancelFunc
}

func read(p *Playback) *reader {
	ctx, cancel := context.WithCancel(context.Background())
	r := &reader{result: make(chan error, 1), cancel: cancel}
	go func() {
		_, err := p.WriteTo(ctx, r)
		r.result <- err
	}()
	return r
}

func (r *reader) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.got.Write(b)
}

func (r *reader) bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.got.Bytes()...)
}

func (r *reader) waitFor(t *testing.T, want []byte) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if bytes.Contains(r.bytes(), want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q; have %d bytes", want, len(r.bytes()))
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func open(t *testing.T, m *Manager, src Source, offset int64) *Playback {
	t.Helper()
	p, err := m.Open(context.Background(), src, offset)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStartSendsHeaderThenEncode(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	p := open(t, m, src, 0)
	defer p.Close()
	if p.Kind != KindStart {
		t.Fatalf("kind = %q", p.Kind)
	}
	r := read(p)
	defer r.cancel()

	frag := testBox("mdat", []byte("first-fragment"))
	f.last().emit(frag)
	r.waitFor(t, []byte("first-fragment"))

	got := r.bytes()
	if !bytes.HasPrefix(got, patchedPrefix(t, f, src)) {
		t.Fatal("stream must begin with ftyp and moov carrying the full duration")
	}
	if !bytes.Contains(got, []byte("sidx")) {
		t.Fatal("stream must carry the segment index")
	}
	if int64(bytes.Index(got, []byte("first-fragment"))) < src.Layout.DataStart {
		t.Fatal("fragment data must come after the fixed header")
	}
	if bytes.Count(got, []byte("moov")) != 1 {
		t.Fatal("a start must not repeat moov")
	}
}

func TestSeekIsMoofFirstAndAtChunkTime(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	first := open(t, m, src, 0)
	r1 := read(first)
	defer r1.cancel()
	r1.waitFor(t, []byte("sidx"))
	firstEncoder := f.last()

	second := open(t, m, src, src.Layout.ChunkOffset(25))
	defer second.Close()
	if second.Kind != KindSeek || second.Seek != 50 {
		t.Fatalf("got kind %q seek %v", second.Kind, second.Seek)
	}
	select {
	case <-firstEncoder.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the previous encoder must exit before the seek starts")
	}
	r2 := read(second)
	defer r2.cancel()
	f.last().emit(testBox("mdat", []byte("after-seek")))
	r2.waitFor(t, []byte("after-seek"))

	got := r2.bytes()
	if bytes.Contains(got, []byte("sidx")) {
		t.Fatal("a second index would replace the player's seek map")
	}
	if bytes.Contains(got, []byte("moov")) {
		t.Fatal("an unchanged moov must not be repeated")
	}
	if f.seeks[len(f.seeks)-1] != 50 {
		t.Fatalf("encoder started at %v, want 50", f.seeks[len(f.seeks)-1])
	}
	if f.prepared != 1 {
		t.Fatalf("encoder chosen %d times, want once for the whole stream", f.prepared)
	}
}

func TestResumeContinuesByteForByte(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	first := open(t, m, src, 0)
	r1 := read(first)
	f.last().emit(testBox("mdat", bytes.Repeat([]byte("A"), 1000)))
	r1.waitFor(t, bytes.Repeat([]byte("A"), 1000))
	received := r1.bytes()
	r1.cancel()
	<-r1.result
	first.Close()

	// A reconnect lands inside the data already sent.
	resumeAt := int64(len(received) - 300)
	second := open(t, m, src, resumeAt)
	defer second.Close()
	if second.Kind != KindResume {
		t.Fatalf("kind = %q, want resume", second.Kind)
	}
	if f.count() != 1 {
		t.Fatalf("resume started %d encoders, want the original one only", f.count())
	}
	r2 := read(second)
	defer r2.cancel()
	f.last().emit(testBox("mdat", []byte("continued")))
	r2.waitFor(t, []byte("continued"))

	got := r2.bytes()
	if !bytes.HasPrefix(got, received[resumeAt:]) {
		t.Fatal("resume must replay exactly the bytes the client is missing")
	}
}

func TestChunkBoundaryInsideBufferIsStillASeek(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	first := open(t, m, src, 0)
	r1 := read(first)
	defer r1.cancel()
	size := int(src.Layout.ChunkSize * 3)
	f.last().emit(testBox("mdat", make([]byte, size)))
	waitFor(t, "buffered data", func() bool { return len(r1.bytes()) > size })

	second := open(t, m, src, src.Layout.ChunkOffset(2))
	defer second.Close()
	if second.Kind != KindSeek {
		t.Fatalf("kind = %q, want seek", second.Kind)
	}
	if f.count() != 2 || f.seeks[1] != 4 {
		t.Fatalf("encodes at %v, want a new one at 4 s", f.seeks)
	}
}

func TestRepeatedSeekSharesEncode(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	offset := src.Layout.ChunkOffset(10)

	first := open(t, m, src, offset)
	first.Close()
	second := open(t, m, src, offset)
	defer second.Close()
	if f.count() != 1 {
		t.Fatalf("encoders = %d, want the seek to reuse the encode that has not moved", f.count())
	}
}

func TestChangedMoovIsSentInBand(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	first := open(t, m, src, 0)
	r1 := read(first)
	defer r1.cancel()
	r1.waitFor(t, []byte("sidx"))

	f.mu.Lock()
	f.prefix = append(testBox("ftyp", []byte("iso5iso6mp41")), testMoov(1, 2000)...)
	f.mu.Unlock()
	second := open(t, m, src, src.Layout.ChunkOffset(4))
	defer second.Close()
	r2 := read(second)
	defer r2.cancel()
	r2.waitFor(t, []byte("moov"))
	if bytes.Contains(r2.bytes(), []byte("sidx")) {
		t.Fatal("a repeated moov must not bring a second index with it")
	}
}

func TestSeekAfterRestartSendsMoov(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	p := open(t, m, src, src.Layout.ChunkOffset(4))
	defer p.Close()
	r := read(p)
	defer r.cancel()
	r.waitFor(t, []byte("moov"))
}

func TestResumeBeforeDataStartContinuesHeader(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	p := open(t, m, src, 100)
	defer p.Close()
	if p.Kind != KindStart {
		t.Fatalf("kind = %q", p.Kind)
	}
	r := read(p)
	defer r.cancel()
	f.last().emit(testBox("mdat", []byte("payload")))
	r.waitFor(t, []byte("payload"))

	header := patchedPrefix(t, f, src)
	got := r.bytes()
	if !bytes.HasPrefix(got, header[100:]) {
		t.Fatal("response must continue the header from the requested byte")
	}
}

func TestUnknownOffsetIsRefusedWithoutDisturbingEncode(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	running := open(t, m, src, 0)
	defer running.Close()
	r := read(running)
	defer r.cancel()
	r.waitFor(t, []byte("sidx"))

	_, err := m.Open(context.Background(), src, src.Layout.ChunkOffset(10)+5)
	if !errors.Is(err, ErrCannotResume) {
		t.Fatalf("err = %v, want ErrCannotResume", err)
	}
	if f.count() != 1 {
		t.Fatalf("encoders = %d; a refused request must not start one", f.count())
	}
	select {
	case <-f.last().killed:
		t.Fatal("a refused request must leave the running encode alone")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCancelledRequestLeavesEncodeAlone(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	running := open(t, m, src, 0)
	defer running.Close()
	r := read(running)
	defer r.cancel()
	r.waitFor(t, []byte("sidx"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Open(ctx, src, src.Layout.ChunkOffset(10)); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if f.count() != 1 {
		t.Fatalf("encoders = %d; a request with no client must not start one", f.count())
	}
}

func TestIdleEncodeStops(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.IdleTimeout = 20 * time.Millisecond

	p := open(t, m, src, 0)
	encoder := f.last()
	p.Close()
	select {
	case <-encoder.killed:
	case <-time.After(5 * time.Second):
		t.Fatal("an encode with no client must stop")
	}
}

func TestReattachCancelsIdleStop(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.IdleTimeout = 100 * time.Millisecond

	first := open(t, m, src, 0)
	encoder := f.last()
	first.Close()
	second := open(t, m, src, 0)
	defer second.Close()
	if f.count() != 1 {
		t.Fatalf("encoders = %d", f.count())
	}
	select {
	case <-encoder.killed:
		t.Fatal("a reconnected encode must keep running")
	case <-time.After(250 * time.Millisecond):
	}
}

func TestDetachedEncodeOfOtherStreamIsReleased(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	other := src
	other.Key = "other"
	m := NewManager(nil)

	first := open(t, m, src, 0)
	firstEncoder := f.last()
	first.Close()

	second := open(t, m, other, 0)
	defer second.Close()
	select {
	case <-firstEncoder.killed:
	case <-time.After(5 * time.Second):
		t.Fatal("a detached encode must give way to a new stream")
	}
}

func TestUnavailableIsNotPinned(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	calls := 0
	src.Prepare = func() (Spawner, error) {
		calls++
		if calls == 1 {
			return nil, ErrUnavailable
		}
		return f.spawn, nil
	}
	m := NewManager(nil)

	if _, err := m.Open(context.Background(), src, 0); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
	p := open(t, m, src, 0)
	defer p.Close()
	if calls != 2 {
		t.Fatalf("Prepare called %d times, want a retry after refusal", calls)
	}
}

// scriptedSource starts encoders that produce fixed output and exit with waitErr.
func scriptedSource(t *testing.T, output []byte, waitErr error, prepared *int) Source {
	t.Helper()
	l, err := NewLayout(100, Options{BitrateMbps: 1})
	if err != nil {
		t.Fatal(err)
	}
	return Source{
		Key:      "file",
		Duration: 100,
		Layout:   l,
		Prepare: func() (Spawner, error) {
			*prepared++
			return func(ctx context.Context, seek float64) (*Process, error) {
				return &Process{
					Stdout: bytes.NewReader(output),
					Wait:   func() error { return waitErr },
				}, nil
			}, nil
		},
	}
}

func TestPlanThatProducesNoMediaIsChosenAgain(t *testing.T) {
	failure := errors.New("no encoder session")
	cases := []struct {
		name   string
		output []byte
	}{
		{"no output", nil},
		{"header only", testPrefix()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prepared := 0
			src := scriptedSource(t, c.output, failure, &prepared)
			m := NewManager(nil)
			offset := src.Layout.ChunkOffset(5)

			first := open(t, m, src, offset)
			var out bytes.Buffer
			if _, err := first.WriteTo(context.Background(), &out); !errors.Is(err, failure) {
				t.Fatalf("WriteTo err = %v, want the encoder failure", err)
			}
			first.Close()

			// The stream loses its failed session before the next attempt arrives.
			m.expire(m.stream(src.Key), first.session)

			second := open(t, m, src, offset)
			defer second.Close()
			if prepared != 2 {
				t.Fatalf("Prepare called %d times, want a fresh choice after a failed encode", prepared)
			}
		})
	}
}

func TestFreshPlaybackChoosesEncoderAgain(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.IdleTimeout = 10 * time.Millisecond

	first := open(t, m, src, 0)
	encoder := f.last()
	first.Close()
	<-encoder.killed
	waitFor(t, "encode to be released", func() bool {
		st := m.stream(src.Key)
		st.mu.Lock()
		defer st.mu.Unlock()
		return st.active == nil
	})

	second := open(t, m, src, 0)
	defer second.Close()
	if f.prepared != 2 {
		t.Fatalf("encoder chosen %d times, want once per playback", f.prepared)
	}
}

func TestFailedEncodeIsReportedToTheClient(t *testing.T) {
	failure := errors.New("exit status 1")
	cases := []struct {
		name    string
		output  []byte
		waitErr error
		want    error
	}{
		{"exit before any output", nil, failure, failure},
		{"exit between boxes", append(testPrefix(), testBox("mdat", []byte("x"))...), failure, failure},
		{"cut inside a box", append(testPrefix(), testBox("mdat", []byte("xyz"))[:10]...), nil, io.ErrUnexpectedEOF},
		{"no output and clean exit", nil, nil, io.ErrUnexpectedEOF},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prepared := 0
			src := scriptedSource(t, c.output, c.waitErr, &prepared)
			m := NewManager(nil)

			p := open(t, m, src, src.Layout.ChunkOffset(3))
			defer p.Close()
			var out bytes.Buffer
			if _, err := p.WriteTo(context.Background(), &out); !errors.Is(err, c.want) {
				t.Fatalf("WriteTo err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestEncoderErrorKeepsItsMessage(t *testing.T) {
	failure := errors.New("OpenEncodeSessionEx failed: out of memory")
	var logged string
	var mu sync.Mutex
	prepared := 0
	src := scriptedSource(t, nil, failure, &prepared)
	m := NewManager(func(format string, args ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		logged = fmt.Sprintf(format, args...)
	})

	p := open(t, m, src, src.Layout.ChunkOffset(3))
	defer p.Close()
	<-p.session.done
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(logged, "out of memory") {
		t.Fatalf("logged %q, want the encoder's own message", logged)
	}
}

func TestMoovRepeatedOnceClientsCopyMayDiffer(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	first := open(t, m, src, 0)
	r1 := read(first)
	defer r1.cancel()
	r1.waitFor(t, []byte("sidx"))

	f.mu.Lock()
	f.prefix = append(testBox("ftyp", []byte("iso5iso6mp41")), testMoov(1, 2000)...)
	f.mu.Unlock()
	changed := open(t, m, src, src.Layout.ChunkOffset(4))
	changed.Close()

	// The same moov again, but the client may have abandoned the response that carried it.
	again := open(t, m, src, src.Layout.ChunkOffset(8))
	defer again.Close()
	r3 := read(again)
	defer r3.cancel()
	r3.waitFor(t, []byte("moov"))
}

func TestStalledClientGivesWayToNewStream(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	other := src
	other.Key = "other"
	m := NewManager(nil)

	stalled := open(t, m, src, 0)
	defer stalled.Close()
	encoder := f.last()
	stalled.session.mu.Lock()
	stalled.session.lastRead = time.Now().Add(-stallRelease - time.Second)
	stalled.session.mu.Unlock()

	second := open(t, m, other, 0)
	defer second.Close()
	select {
	case <-encoder.killed:
	case <-time.After(5 * time.Second):
		t.Fatal("an encode whose client stopped reading must give way to a new stream")
	}
}

func TestActiveClientKeepsEncodeWhenOtherStreamStarts(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	other := src
	other.Key = "other"
	m := NewManager(nil)

	first := open(t, m, src, 0)
	defer first.Close()
	encoder := f.last()
	second := open(t, m, other, 0)
	defer second.Close()
	select {
	case <-encoder.killed:
		t.Fatal("an encode with an active client must not be stopped")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEncodedDataSurvivesEncoderExit(t *testing.T) {
	src := Source{Key: "file", Duration: 100}
	var err error
	if src.Layout, err = NewLayout(100, Options{BitrateMbps: 1}); err != nil {
		t.Fatal(err)
	}
	stream := append(testPrefix(), testBox("mdat", []byte("last-fragment"))...)
	src.Prepare = func() (Spawner, error) {
		return func(ctx context.Context, seek float64) (*Process, error) {
			return &Process{Stdout: bytes.NewReader(stream), Wait: func() error { return nil }}, nil
		}, nil
	}
	m := NewManager(nil)

	p := open(t, m, src, 0)
	defer p.Close()
	var out bytes.Buffer
	if _, err := p.WriteTo(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("last-fragment")) {
		t.Fatal("the end of the encode must reach the client")
	}
}

func TestKeepAliveOnlyAtBoxBoundary(t *testing.T) {
	s := newEncodeSession(0, func() {})
	before := len(s.buf)
	s.lastWrite = time.Now().Add(-time.Minute)

	s.BoxStart()
	ctx, cancel := context.WithCancel(context.Background())
	go s.keepAlive(ctx)
	time.Sleep(keepAliveInterval + keepAliveInterval/2)
	s.mu.Lock()
	inBox := len(s.buf)
	s.mu.Unlock()
	if inBox != before {
		cancel()
		t.Fatal("no box may be inserted while another is being written")
	}

	s.BoxEnd()
	waitFor(t, "keep-alive box", func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return len(s.buf) > before
	})
	cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if string(s.buf[before+4:before+8]) != "free" {
		t.Fatalf("keep-alive was %q", s.buf[before+4:before+8])
	}
}

func TestBufferHoldsEncoderBackAndKeepsResumeWindow(t *testing.T) {
	s := newEncodeSession(1000, func() {})
	gen, ok := s.attach(1000)
	if !ok {
		t.Fatal("attach failed")
	}
	chunk := make([]byte, ringCapacity/2)
	if _, err := s.Write(chunk); err != nil {
		t.Fatal(err)
	}

	blocked := make(chan error, 1)
	go func() {
		_, err := s.Write(make([]byte, ringCapacity))
		blocked <- err
	}()
	select {
	case <-blocked:
		t.Fatal("output must wait for a client that has not read")
	case <-time.After(50 * time.Millisecond):
	}

	dst := make([]byte, ringCapacity/2)
	pos := int64(1000)
	for pos < 1000+ringCapacity {
		n, err := s.readAt(context.Background(), gen, pos, dst)
		if err != nil {
			t.Fatal(err)
		}
		pos += int64(n)
	}

	s.mu.Lock()
	base := s.base
	s.mu.Unlock()
	if pos-base < keepBehind {
		t.Fatalf("only %d bytes kept behind the reader, want at least %d", pos-base, keepBehind)
	}
	s.stop2()
	<-blocked
}

// stop2 ends a session constructed without a pump.
func (s *encodeSession) stop2() {
	s.mu.Lock()
	s.stopped = true
	s.cond.Broadcast()
	s.mu.Unlock()
}
