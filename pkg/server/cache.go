package server

import (
	"net/http"

	"github.com/fcjr/aia-transport-go"
)

type ForceCacheTransport struct {
	Transport http.RoundTripper
}

// RoundTrip transport function that will force a Cache-Control of 5 years
// on all HTTP 2xx responses, so that httpcache used by imageproxy will continue
// to handle the cache as fresh, even when no cache header is set by upstream
// server.
func (s *ForceCacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Perform original request
	resp, err := s.Transport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	// Overwrite cache behavior on 2xx responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		resp.Header.Set("Cache-Control", "public, max-age=157680000")
	}

	return resp, nil
}

func NewForceCacheTransport() *ForceCacheTransport {
	fct := new(ForceCacheTransport)

	// this is what willnorris.com/go/imageproxy does by default,
	// so keep the same here
	fct.Transport, _ = aia.NewTransport()

	return fct
}
