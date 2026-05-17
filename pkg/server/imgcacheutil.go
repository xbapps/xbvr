package server

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fcjr/aia-transport-go"
	"golang.org/x/image/webp"

	"github.com/xbapps/xbvr/pkg/config"
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

	if config.Config.Cache.ConvertWebPToJPEG {
		convertWebPResponseToJPEG(resp)
	}

	// Overwrite cache behavior on 2xx responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Force cache duration in the diskCache to 5 years
		resp.Header.Set("Cache-Control", "public, max-age=157680000")
	}

	return resp, nil
}

func convertWebPResponseToJPEG(resp *http.Response) {
	if resp == nil || resp.Body == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || !isWebPResponse(resp) {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	resp.Body.Close()

	img, err := webp.Decode(bytes.NewReader(body))
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		return
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, flattenImage(img), &jpeg.Options{Quality: 90}); err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		return
	}

	resp.Body = io.NopCloser(bytes.NewReader(out.Bytes()))
	resp.ContentLength = int64(out.Len())
	resp.Header.Set("Content-Length", strconv.Itoa(out.Len()))
	resp.Header.Set("Content-Type", "image/jpeg")
	resp.Header.Del("Content-Encoding")
}

func isWebPResponse(resp *http.Response) bool {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "image/webp") {
		return true
	}
	if resp.Request != nil && resp.Request.URL != nil {
		ext := strings.ToLower(filepath.Ext(resp.Request.URL.Path))
		return ext == ".webp"
	}
	return false
}

func flattenImage(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Over)
	return dst
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
