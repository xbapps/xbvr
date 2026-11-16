package transcode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// ringCapacity bounds the encoded bytes held in memory for one stream.
	ringCapacity = 48 << 20

	// keepBehind is how far behind the client's read position bytes stay available. Bytes
	// handed to the socket may never have arrived, so a reconnect can ask for them again.
	keepBehind = 16 << 20

	// compactMinimum avoids moving the buffer for every small read once it is full.
	compactMinimum = 8 << 20

	// keepAliveInterval is the longest a client waits between bytes while an encode has
	// nothing to send, which keeps player read timeouts from firing.
	keepAliveInterval = 2 * time.Second
	keepAliveBoxSize  = 16

	// defaultIdleTimeout is how long an encode outlives its client, so that a reconnect can
	// continue it.
	defaultIdleTimeout = 60 * time.Second

	// stallRelease is how long a client may go without reading before its encode can be
	// reclaimed for another stream. A client that vanished keeps its socket write blocked
	// until TCP gives up, so it never detaches by itself.
	stallRelease = 2 * time.Minute

	// defaultPinLifetime is how long a stream's encoder choice and header are remembered
	// once nothing is using them.
	defaultPinLifetime = 30 * time.Minute

	stopWait       = 5 * time.Second
	playbackBuffer = 256 << 10
)

var (
	// ErrUnavailable means no encoder can currently take this stream.
	ErrUnavailable = errors.New("transcode: no encoder available")

	// ErrCannotResume means a request continues a stream at a byte this server can no longer
	// produce. A client holding parser state at that byte cannot use a different encode.
	ErrCannotResume = errors.New("transcode: offset cannot be resumed")
)

var (
	errSessionStopped = errors.New("transcode: encode stopped")
	errSuperseded     = errors.New("transcode: client replaced by a newer request")
	errEvicted        = errors.New("transcode: position no longer buffered")
)

// Process is a running encoder. Wait must only be called once Stdout has been read to its end.
type Process struct {
	Stdout io.Reader
	Wait   func() error
}

// Spawner starts an encoder at a source time. Cancelling ctx must end the encoder.
type Spawner func(ctx context.Context, seekSeconds float64) (*Process, error)

// Source describes one stream. Prepare chooses the encoder and is only called when the stream
// has no choice pinned yet.
type Source struct {
	Key      string
	Duration float64
	Layout   Layout
	Prepare  func() (Spawner, error)
}

// Kinds of request, as reported by Playback.Kind.
const (
	KindStart  = "start"  // from the beginning of the stream
	KindSeek   = "seek"   // at a chunk boundary from the index
	KindResume = "resume" // a reconnect continuing the running encode
)

// Manager runs live encodes and decides how each request maps onto them. A stream has at most
// one encode running; a seek replaces it, while a reconnect continues it byte for byte.
type Manager struct {
	IdleTimeout time.Duration
	PinLifetime time.Duration
	Logf        func(format string, args ...interface{})

	mu      sync.Mutex
	streams map[string]*stream
}

// NewManager returns a Manager with the default timeouts.
func NewManager(logf func(format string, args ...interface{})) *Manager {
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	return &Manager{
		IdleTimeout: defaultIdleTimeout,
		PinLifetime: defaultPinLifetime,
		Logf:        logf,
		streams:     map[string]*stream{},
	}
}

// stream is the per-file, per-codec state that outlives any single encode.
type stream struct {
	mu         sync.Mutex // serialises opens and encode replacement
	spawn      Spawner
	active     *encodeSession
	planFailed atomic.Bool // the encoder choice produced no media and should be made again
	lastUsed   time.Time   // guarded by Manager.mu

	headerMu    sync.Mutex
	prefix      []byte
	header      []byte
	headerReady chan struct{}
	sendMoov    bool // every encode must repeat its moov, as the client's copy may differ
}

func (m *Manager) stream(key string) *stream {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, st := range m.streams {
		if k == key || now.Sub(st.lastUsed) <= m.PinLifetime || !st.mu.TryLock() {
			continue
		}
		if st.active == nil {
			delete(m.streams, k)
		}
		st.mu.Unlock()
	}
	st := m.streams[key]
	if st == nil {
		st = &stream{headerReady: make(chan struct{})}
		m.streams[key] = st
	}
	st.lastUsed = now
	return st
}

// Open attaches a request at byte offset to an encode, starting one if the request needs it.
// A request whose context has already ended leaves the running encode alone. The returned
// Playback must be closed.
func (m *Manager) Open(ctx context.Context, src Source, offset int64) (*Playback, error) {
	l := src.Layout
	st := m.stream(src.Key)
	st.mu.Lock()
	defer st.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	position := offset
	if position < l.DataStart {
		position = l.DataStart
	}
	chunk, exact := l.ChunkAt(position)
	active := st.active

	var kind string
	var gen uint64
	attached := false
	switch {
	case offset < l.DataStart:
		kind = KindStart
		if active != nil && active.start == l.DataStart {
			gen, attached = active.attach(position)
		}
		if active == nil {
			st.beginPlayback()
		}
	case exact:
		kind = KindSeek
		if active != nil && active.start == position {
			gen, attached = active.attach(position)
		}
	default:
		kind = KindResume
		if active != nil {
			gen, attached = active.attach(position)
		}
		if !attached {
			return nil, ErrCannotResume
		}
	}

	if !attached {
		session, sessionGen, err := m.replace(st, src, chunk, position)
		if err != nil {
			return nil, err
		}
		active, gen = session, sessionGen
	}

	return &Playback{
		Kind:    kind,
		Chunk:   chunk,
		Seek:    l.StartSeconds(chunk),
		manager: m,
		stream:  st,
		session: active,
		gen:     gen,
		offset:  offset,
		pos:     position,
		layout:  l,
	}, nil
}

// replace stops the current encode and starts another with the request already attached, so
// a failure of the new encoder is always delivered to it. The previous encoder has exited
// before the new one starts, so its GPU memory is available again.
func (m *Manager) replace(st *stream, src Source, chunk int, position int64) (*encodeSession, uint64, error) {
	if previous := st.active; previous != nil {
		previous.stop()
		st.active = nil
	}
	m.stopDetached(st)

	if st.planFailed.Swap(false) {
		st.spawn = nil
	}
	if st.spawn == nil {
		spawn, err := src.Prepare()
		if err != nil {
			return nil, 0, err
		}
		st.spawn = spawn
	}

	seek := src.Layout.StartSeconds(chunk)
	ctx, cancel := context.WithCancel(context.Background())
	process, err := st.spawn(ctx, seek)
	if err != nil {
		cancel()
		return nil, 0, err
	}

	session := newEncodeSession(position, cancel)
	gen, _ := session.attach(position)
	st.active = session
	go session.pump(m, st, src, process, seek)
	go session.keepAlive(ctx)
	return session, gen, nil
}

// stopDetached ends encodes no client is using, releasing their encoder sessions. Streams
// busy with another request are skipped rather than waited on.
func (m *Manager) stopDetached(except *stream) {
	m.mu.Lock()
	others := make([]*stream, 0, len(m.streams))
	for _, st := range m.streams {
		if st != except {
			others = append(others, st)
		}
	}
	m.mu.Unlock()

	for _, st := range others {
		if !st.mu.TryLock() {
			continue
		}
		if st.active != nil && st.active.reclaimable() {
			st.active.stop()
			st.active = nil
		}
		st.mu.Unlock()
	}
}

func (m *Manager) expire(st *stream, session *encodeSession) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.active != session || session.isAttached() {
		return
	}
	session.stop()
	st.active = nil
}

// beginPlayback starts a new viewing of the stream. The encoder is chosen again, since free
// GPU memory may have changed, and earlier format changes no longer apply.
func (st *stream) beginPlayback() {
	st.spawn = nil
	st.headerMu.Lock()
	st.sendMoov = false
	st.headerMu.Unlock()
}

// adoptHeader records an encode's ftyp and moov and reports whether they must also be sent
// inside the stream. A client joining part-way through the file never saw the fixed header,
// and one holding a different moov needs the new one. Once either has happened every later
// encode repeats its moov, because a response can be abandoned before the client reads it.
func (st *stream) adoptHeader(l Layout, prefix []byte, midStream bool) (bool, error) {
	st.headerMu.Lock()
	defer st.headerMu.Unlock()
	if st.header != nil && bytes.Equal(st.prefix, prefix) {
		return st.sendMoov, nil
	}
	header, err := l.Header(prefix)
	if err != nil {
		return false, err
	}
	first := st.header == nil
	st.prefix = append([]byte(nil), prefix...)
	st.header = header
	if first {
		close(st.headerReady)
	}
	if !first || midStream {
		st.sendMoov = true
	}
	return st.sendMoov, nil
}

func (st *stream) cachedHeader() []byte {
	st.headerMu.Lock()
	defer st.headerMu.Unlock()
	return st.header
}

// encodeSession holds one running encode and a window of its output. Positions are absolute
// stream offsets, so a reconnect names the exact byte it needs.
type encodeSession struct {
	start  int64
	cancel context.CancelFunc
	done   chan struct{}

	mu         sync.Mutex
	cond       *sync.Cond
	buf        []byte
	base       int64
	readPos    int64
	gen        uint64
	attached   bool
	atBoundary bool
	lastWrite  time.Time
	lastRead   time.Time
	stopped    bool
	ended      bool
	err        error
	inMedia    bool // past the header, in the encoder's fragments
	output     bool // at least one media box was produced
	idle       *time.Timer
}

func newEncodeSession(start int64, cancel context.CancelFunc) *encodeSession {
	s := &encodeSession{
		start:      start,
		base:       start,
		readPos:    start,
		cancel:     cancel,
		done:       make(chan struct{}),
		atBoundary: true,
		lastWrite:  time.Now(),
		lastRead:   time.Now(),
	}
	s.cond = sync.NewCond(&s.mu)
	s.buf = append(s.buf, freeBox(keepAliveBoxSize)...)
	return s
}

// pump moves encoder output into the buffer until the encoder ends.
func (s *encodeSession) pump(m *Manager, st *stream, src Source, process *Process, seek float64) {
	defer close(s.done)
	defer s.cancel()

	streamErr := s.copyOutput(st, src, process, seek)
	if streamErr != nil {
		s.cancel()
	}
	waitErr := process.Wait()

	s.mu.Lock()
	stopped := s.stopped
	s.ended = true
	s.err = endError(streamErr, waitErr)
	failure, producedMedia := s.err, s.output
	s.cond.Broadcast()
	s.mu.Unlock()

	if failure != nil && !stopped {
		if !producedMedia {
			st.planFailed.Store(true)
		}
		m.Logf("transcode %s: encode at %.3f ended: %v", src.Key, seek, failure)
	}
}

// endError combines how the output ended with how the encoder exited. The encoder's error
// comes first because it names the cause; the output error only records the symptom.
func endError(streamErr, waitErr error) error {
	switch {
	case waitErr == nil:
		return streamErr
	case streamErr == nil:
		return waitErr
	}
	return fmt.Errorf("%w (%v)", waitErr, streamErr)
}

// copyOutput moves encoder output into the buffer. A nil result means the encoder's output
// ended exactly on a box boundary; an output that stops anywhere else is an error, never a
// plain EOF that could be mistaken for the end of the video.
func (s *encodeSession) copyOutput(st *stream, src Source, process *Process, seek float64) error {
	err := s.copyBoxes(st, src, process, seek)
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

func (s *encodeSession) copyBoxes(st *stream, src Source, process *Process, seek float64) error {
	prefix, err := ReadMP4Header(process.Stdout, src.Duration)
	if err != nil {
		return err
	}
	inBand, err := st.adoptHeader(src.Layout, prefix, seek > 0)
	if err != nil {
		return err
	}
	if inBand {
		s.BoxStart()
		if _, err := s.Write(prefix); err != nil {
			return err
		}
		s.BoxEnd()
	}
	s.mu.Lock()
	s.inMedia = true
	s.mu.Unlock()
	return PatchFragmentTimes(process.Stdout, s, prefix, seek)
}

// Write adds encoder output to the buffer, waiting when the client has not caught up. The
// encoder is thereby held back to the pace the client reads at.
func (s *encodeSession) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := len(p)
	for len(p) > 0 {
		if s.stopped {
			return total - len(p), errSessionStopped
		}
		s.compact()
		room := ringCapacity - len(s.buf)
		if room <= 0 {
			s.cond.Wait()
			continue
		}
		n := len(p)
		if n > room {
			n = room
		}
		s.buf = append(s.buf, p[:n]...)
		p = p[n:]
		s.lastWrite = time.Now()
		s.cond.Broadcast()
	}
	return total, nil
}

// BoxStart marks the stream as inside a top-level box.
func (s *encodeSession) BoxStart() {
	s.mu.Lock()
	s.atBoundary = false
	s.mu.Unlock()
}

// BoxEnd marks the stream as between top-level boxes.
func (s *encodeSession) BoxEnd() {
	s.mu.Lock()
	s.atBoundary = true
	if s.inMedia {
		s.output = true
	}
	s.mu.Unlock()
}

// compact discards data the client can no longer ask for, once the buffer is full.
func (s *encodeSession) compact() {
	if len(s.buf) < ringCapacity {
		return
	}
	drop := s.readPos - keepBehind - s.base
	if drop < compactMinimum {
		return
	}
	n := copy(s.buf, s.buf[drop:])
	s.buf = s.buf[:n]
	s.base += drop
}

// keepAlive inserts a skippable box whenever the encoder has been silent at a box boundary.
func (s *encodeSession) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(keepAliveInterval / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		s.mu.Lock()
		idle := time.Since(s.lastWrite) >= keepAliveInterval
		if idle && s.atBoundary && !s.ended && !s.stopped && len(s.buf)+keepAliveBoxSize <= ringCapacity {
			s.buf = append(s.buf, freeBox(keepAliveBoxSize)...)
			s.lastWrite = time.Now()
			s.cond.Broadcast()
		}
		s.mu.Unlock()
	}
}

// attach makes a client the reader of this encode at pos, displacing any previous client.
func (s *encodeSession) attach(pos int64) (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	end := s.base + int64(len(s.buf))
	if s.stopped || (s.ended && s.err != nil) || pos < s.base || pos > end {
		return 0, false
	}
	s.gen++
	s.attached = true
	s.readPos = pos
	s.lastRead = time.Now()
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	s.cond.Broadcast()
	return s.gen, true
}

// detach releases a client. Without a replacement the encode is stopped after timeout.
func (s *encodeSession) detach(gen uint64, timeout time.Duration, onIdle func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gen != gen || !s.attached {
		return
	}
	s.attached = false
	if !s.stopped {
		s.idle = time.AfterFunc(timeout, onIdle)
	}
}

func (s *encodeSession) isAttached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attached
}

// reclaimable reports whether no client is making use of this encode.
func (s *encodeSession) reclaimable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.attached || time.Since(s.lastRead) > stallRelease
}

// readAt copies buffered bytes at pos into dst, waiting for more if necessary. Bytes the
// encoder finished are still delivered after it exits.
func (s *encodeSession) readAt(ctx context.Context, gen uint64, pos int64, dst []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		switch {
		case ctx.Err() != nil:
			return 0, ctx.Err()
		case s.gen != gen:
			return 0, errSuperseded
		case s.stopped:
			return 0, errSessionStopped
		case pos < s.base:
			return 0, errEvicted
		}
		if end := s.base + int64(len(s.buf)); pos < end {
			n := copy(dst, s.buf[pos-s.base:])
			s.readPos = pos + int64(n)
			s.lastRead = time.Now()
			s.cond.Broadcast()
			return n, nil
		}
		if s.ended {
			if s.err != nil {
				return 0, s.err
			}
			return 0, io.EOF
		}
		s.cond.Wait()
	}
}

func (s *encodeSession) wake() {
	s.mu.Lock()
	s.cond.Broadcast()
	s.mu.Unlock()
}

// stop ends the encode and returns once its encoder has exited.
func (s *encodeSession) stop() {
	s.mu.Lock()
	s.stopped = true
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	s.cond.Broadcast()
	s.mu.Unlock()

	s.cancel()
	select {
	case <-s.done:
	case <-time.After(stopWait):
	}
}

// Playback is one client request attached to an encode.
type Playback struct {
	Kind  string
	Chunk int
	Seek  float64

	manager *Manager
	stream  *stream
	session *encodeSession
	gen     uint64
	offset  int64
	pos     int64
	layout  Layout
}

// WriteTo sends the stream to w, beginning with whatever part of the fixed header the request
// covers, until the encode ends, the client goes away or a newer request replaces this one.
func (p *Playback) WriteTo(ctx context.Context, w io.Writer) (int64, error) {
	stopWaking := context.AfterFunc(ctx, p.session.wake)
	defer stopWaking()

	var sent int64
	if p.offset < p.layout.DataStart {
		header, err := p.waitForHeader(ctx)
		if err != nil {
			return sent, err
		}
		n, err := w.Write(header[p.offset:])
		sent += int64(n)
		if err != nil {
			return sent, err
		}
	}

	buffer := make([]byte, playbackBuffer)
	for {
		n, err := p.session.readAt(ctx, p.gen, p.pos, buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return sent, nil
			}
			return sent, err
		}
		written, err := w.Write(buffer[:n])
		sent += int64(written)
		p.pos += int64(written)
		if err != nil {
			return sent, err
		}
	}
}

func (p *Playback) waitForHeader(ctx context.Context) ([]byte, error) {
	select {
	case <-p.stream.headerReady:
		return p.stream.cachedHeader(), nil
	case <-p.session.done:
		// The encode may have finished immediately after publishing its header.
		if header := p.stream.cachedHeader(); header != nil {
			return header, nil
		}
		if err := p.session.finalError(); err != nil {
			return nil, err
		}
		return nil, errSessionStopped
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *encodeSession) finalError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close detaches the client, leaving the encode running briefly so a reconnect can continue.
func (p *Playback) Close() {
	p.session.detach(p.gen, p.manager.IdleTimeout, func() {
		p.manager.expire(p.stream, p.session)
	})
}
