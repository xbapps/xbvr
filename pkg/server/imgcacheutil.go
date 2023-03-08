package server

import (
	"net/http"

	"github.com/fcjr/aia-transport-go"
)

// Change INCOMING response header's Cache-Control for persistent disk cache
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
		// Force cache duration in the diskCache to 5 years
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

// Change OUTGOING response header cache control, so that VR client
// will not cache as long as we do. This helps refresh the data in the
// VR client after a user has wiped the disk cache in xbvr.
type CacheHeaderResponseWriter struct {
	http.ResponseWriter
}

func (w *CacheHeaderResponseWriter) WriteHeader(statusCode int) {
	if statusCode >= 200 && statusCode < 300 {
		// Force cache duration for VR client to 1 day
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func ForceShortCacheHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&CacheHeaderResponseWriter{w}, r)
	})
}
