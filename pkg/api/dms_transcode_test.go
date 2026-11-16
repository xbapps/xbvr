package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestParseByteRange(t *testing.T) {
	cases := []struct {
		header     string
		total      int64
		start, end int64
		ok         bool
	}{
		{"", 1000, 0, 0, false},
		{"bytes=0-", 1000, 0, 999, true},
		{"bytes=100-199", 1000, 100, 199, true},
		{"bytes=900-10000", 1000, 900, 999, true},
		{"bytes=-100", 1000, 900, 999, true},
		{"bytes=-5000", 1000, 0, 999, true},
		{"bytes=1000-", 1000, 0, 0, false},
		{"bytes=100-99", 1000, 0, 0, false},
		{"bytes=5", 1000, 0, 0, false},
		{"items=0-1", 1000, 0, 0, false},
		{"bytes=0-", 0, 0, 0, false},
	}
	for _, c := range cases {
		start, end, ok := parseByteRange(c.header, c.total)
		if ok != c.ok || start != c.start || end != c.end {
			t.Errorf("parseByteRange(%q, %d) = (%d, %d, %v), want (%d, %d, %v)",
				c.header, c.total, start, end, ok, c.start, c.end, c.ok)
		}
	}
}

func TestRangeBeyondEnd(t *testing.T) {
	cases := []struct {
		header string
		want   bool
	}{
		{"", false},
		{"bytes=99-", false},
		{"bytes=100-", true},
		{"bytes=200-300", true},
		{"bytes=-5", false},
		{"items=100-", false},
	}
	for _, c := range cases {
		if got := rangeBeyondEnd(c.header, 100); got != c.want {
			t.Errorf("rangeBeyondEnd(%q) = %v, want %v", c.header, got, c.want)
		}
	}
}

func TestStreamFailed(t *testing.T) {
	live := context.Background()
	gone, cancel := context.WithCancel(live)
	cancel()
	failure := errors.New("exit status 1")
	cases := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{"finished", live, nil, false},
		{"encoder failed", live, failure, true},
		{"client left", gone, failure, false},
		{"client left cleanly", gone, nil, false},
	}
	for _, c := range cases {
		if got := streamFailed(c.ctx, c.err); got != c.want {
			t.Errorf("%s: streamFailed = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTranscodeClient(t *testing.T) {
	cases := []struct {
		name    string
		remote  string
		headers map[string]string
		want    string
	}{
		{"direct", "192.168.1.20:5000", nil, "192.168.1.20"},
		{"real ip wins", "10.0.0.1:443", map[string]string{"X-Real-IP": "192.168.1.30", "X-Forwarded-For": "192.168.1.40"}, "192.168.1.30"},
		{"first forwarded hop", "10.0.0.1:443", map[string]string{"X-Forwarded-For": "192.168.1.40, 10.0.0.2"}, "192.168.1.40"},
		{"unparseable", "pipe", nil, "pipe"},
	}
	for _, c := range cases {
		req := &http.Request{RemoteAddr: c.remote, Header: http.Header{}}
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
		if got := transcodeClient(req); got != c.want {
			t.Errorf("%s: transcodeClient = %q, want %q", c.name, got, c.want)
		}
	}
}
