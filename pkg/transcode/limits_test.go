package transcode

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// keyed returns src under another stream key, as another file, codec or client would be.
func keyed(src Source, key string) Source {
	src.Key = key
	return src
}

func TestEncodeLimitRefusesAnotherStream(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 2 }

	first := open(t, m, keyed(src, "a"), 0)
	defer first.Close()
	second := open(t, m, keyed(src, "b"), 0)
	defer second.Close()
	if _, err := m.Open(context.Background(), keyed(src, "c"), 0); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy with two clients reading", err)
	}
	if f.count() != 2 {
		t.Fatalf("encoders = %d, want 2", f.count())
	}
}

func TestEncodeLimitDefaultsWhenUnset(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 0 }

	for i := 0; i < DefaultMaxEncodes; i++ {
		p := open(t, m, keyed(src, string(rune('a'+i))), 0)
		defer p.Close()
	}
	if _, err := m.Open(context.Background(), keyed(src, "over"), 0); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy beyond DefaultMaxEncodes", err)
	}
}

func TestSeekWithinStreamFitsUnderLimitOfOne(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 1 }

	p := open(t, m, src, 0)
	defer p.Close()
	seek := open(t, m, src, src.Layout.ChunkOffset(10))
	defer seek.Close()
	if f.count() != 2 {
		t.Fatalf("encoders = %d; a seek replaces its stream's encode rather than adding one", f.count())
	}
}

func TestDetachedEncodeMakesRoomUnderLimit(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 1 }

	first := open(t, m, keyed(src, "a"), 0)
	firstEncoder := f.last()
	first.Close()

	second, err := m.Open(context.Background(), keyed(src, "b"), 0)
	if err != nil {
		t.Fatalf("err = %v; an encode no client reads must give way", err)
	}
	defer second.Close()
	select {
	case <-firstEncoder.killed:
	case <-time.After(5 * time.Second):
		t.Fatal("the detached encode was not stopped")
	}
}

func TestFailedStartDoesNotHoldASlot(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 1 }

	f.fail = errors.New("no ffmpeg")
	if _, err := m.Open(context.Background(), keyed(src, "a"), 0); err == nil {
		t.Fatal("expected the spawn error")
	}
	f.fail = nil
	p, err := m.Open(context.Background(), keyed(src, "b"), 0)
	if err != nil {
		t.Fatalf("err = %v; a start that failed must release its slot", err)
	}
	p.Close()
}

func TestEndedEncodeReleasesItsSlot(t *testing.T) {
	src := Source{Key: "a", Duration: 100}
	var err error
	if src.Layout, err = NewLayout(100, Options{BitrateMbps: 1}); err != nil {
		t.Fatal(err)
	}
	stream := append(testPrefix(), testBox("mdat", []byte("all"))...)
	src.Prepare = func() (Spawner, error) {
		return func(ctx context.Context, seek float64) (*Process, error) {
			return &Process{Stdout: bytes.NewReader(stream), Wait: func() error { return nil }}, nil
		}, nil
	}
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 1 }

	p := open(t, m, src, 0)
	if _, err := p.WriteTo(context.Background(), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the finished encode's slot", func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.running == 0
	})
	p.Close()
}

func TestConcurrentStartsNeverExceedLimit(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.MaxEncodes = func() int { return 3 }

	var wg sync.WaitGroup
	var mu sync.Mutex
	var opened []*Playback
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p, err := m.Open(context.Background(), keyed(src, string(rune('a'+i))), 0)
			if err == nil {
				mu.Lock()
				opened = append(opened, p)
				mu.Unlock()
			} else if !errors.Is(err, ErrBusy) {
				t.Errorf("err = %v", err)
			}
		}(i)
	}
	wg.Wait()
	defer func() {
		for _, p := range opened {
			p.Close()
		}
	}()
	if len(opened) != 3 || f.count() != 3 {
		t.Fatalf("opened %d streams with %d encoders; want exactly 3", len(opened), f.count())
	}
}

func TestClientLeavingDuringStartLetsEncodeIdleOut(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)
	m.IdleTimeout = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	src.Prepare = func() (Spawner, error) {
		cancel() // the client hangs up while the encoder is being chosen
		return f.prepare()
	}
	if _, err := m.Open(ctx, src, src.Layout.ChunkOffset(10)); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	select {
	case <-f.last().killed:
	case <-time.After(5 * time.Second):
		t.Fatal("an encode whose client left during start must stop after the idle timeout")
	}
}

func TestHeaderFromAnotherLayoutIsAnError(t *testing.T) {
	f := &fakeEncoders{}
	src := newTestSource(t, f)
	m := NewManager(nil)

	p := open(t, m, src, 0)
	r := read(p)
	r.waitFor(t, []byte("sidx"))
	r.cancel()
	<-r.result
	p.Close()

	longer := src
	var err error
	if longer.Layout, err = NewLayout(10000, Options{BitrateMbps: 1, AudioKbps: 128}); err != nil {
		t.Fatal(err)
	}
	longer.Duration = 10000
	p2 := open(t, m, longer, src.Layout.DataStart+100)
	defer p2.Close()
	if _, err := p2.WriteTo(context.Background(), &bytes.Buffer{}); !errors.Is(err, errHeaderMismatch) {
		t.Fatalf("err = %v, want errHeaderMismatch", err)
	}
}
