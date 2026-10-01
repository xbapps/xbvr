package scrape

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xbapps/xbvr/pkg/config"
)

// A resty client built while ScraperProxy is set must send requests
// through the proxy (regression: resty engines silently bypassed the
// setting that colly collectors honored).
func TestNewRestyClientUsesProxy(t *testing.T) {
	old := config.Config.Advanced.ScraperProxy
	t.Cleanup(func() { config.Config.Advanced.ScraperProxy = old })

	var sawHost string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	config.Config.Advanced.ScraperProxy = proxy.URL
	resp, err := NewRestyClient().R().Get("http://example.com/scene")
	if err != nil {
		t.Fatalf("proxied request failed: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("proxied request status = %d, want 200 from the fake proxy", resp.StatusCode())
	}
	if sawHost != "example.com" {
		t.Errorf("proxy saw host %q, want the request routed through it to example.com", sawHost)
	}
}
