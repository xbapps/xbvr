package llmscrape

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBraveSearchRetriesRateLimitAndParses(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "key" {
			t.Errorf("missing API key header")
		}
		if r.URL.Query().Get("safesearch") != "off" {
			t.Errorf("safesearch = %q, want off", r.URL.Query().Get("safesearch"))
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"web": map[string]any{"results": []any{
			map[string]any{"title": "Sweet <strong>Pink</strong> Bunny", "url": "https://a/1", "description": "d"},
		}}})
	}))
	defer srv.Close()

	b := NewBraveSearch("key")
	b.endpoint = srv.URL
	b.minGap = 10 * time.Millisecond
	res, err := b.Search(context.Background(), "q", 5)
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("expected one retry after 429, got %d calls", calls)
	}
	if len(res) != 1 || res[0].Title != "Sweet Pink Bunny" || res[0].URL != "https://a/1" {
		t.Errorf("results = %+v", res)
	}
}

func TestBraveSearchSpacesRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"web":{"results":[]}}`))
	}))
	defer srv.Close()

	b := NewBraveSearch("key")
	b.endpoint = srv.URL
	b.minGap = 150 * time.Millisecond
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, err := b.Search(context.Background(), "q", 1); err != nil {
			t.Fatal(err)
		}
	}
	if el := time.Since(start); el < 280*time.Millisecond {
		t.Errorf("3 requests took %v; they should be spaced at least 150ms apart", el)
	}
}

func TestBraveSearchNeedsKey(t *testing.T) {
	if _, err := NewBraveSearch("").Search(context.Background(), "q", 1); err == nil {
		t.Error("expected an error without an API key")
	}
}

func TestCleanTerms(t *testing.T) {
	cases := map[string]string{
		"Dezyred_Adriana Chechik_Slutty House_Fantasy-Fucked_4096p_8K_LR_180.mp4": "Dezyred Adriana Chechik Slutty House Fantasy Fucked",
		"drvr-katrina lunk-welcome-home-sir-7k_180_LR.mp4":                        "drvr katrina lunk welcome home sir",
		"FPVR-AliciaWilliams-DannySteele-180-POV_8K_UHD.mp4":                      "FPVR Alicia Williams Danny Steele POV",
		"Dezyred_Pussy_Master_4096p_8K_LR_180.mp4.mp4":                            "Dezyred Pussy Master",
		"SLR_ThroattleVR_Fucking My Pornstar Girlfriend_4096p_84265_LR.mp4":       "SLR ThroattleVR Fucking My Pornstar Girlfriend",
		"vac-ddfnvr180109kqsdq-2160.mp4":                                          "vac 2160",
		// folder names
		"DDFNetworkVR.18.01.09.Kira.Queen.Sugar.Daddy.for.the.Queen.XXX.VR180.2160p.MP4-VACCiNE": "DDFNetworkVR 18 01 09 Kira Queen Sugar Daddy for the Queen",
		"Haley Spades, Remy Rune - Seductive Science - Ember Moans":                              "Haley Spades Remy Rune Seductive Science Ember Moans",
	}
	for in, want := range cases {
		if got := CleanTerms(in); got != want {
			t.Errorf("CleanTerms(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestBuildQuery(t *testing.T) {
	file := &FileInfo{Filename: "cleo-vega-sweet-pink-bunny.mp4", Context: "EthernalVR"}
	if q := BuildQuery(file); q != "EthernalVR cleo vega sweet pink bunny VR" {
		t.Errorf("query = %q", q)
	}

	// A scene's own folder is used, without repeating words the filename also has.
	file = &FileInfo{
		Filename:      "Dezyred_Seductive+Science+-+Ember+Moans_4096p_8K_LR_180.mp4",
		Folder:        "Haley Spades, Remy Rune - Seductive Science - Ember Moans",
		FolderIsScene: true,
	}
	if q := BuildQuery(file); q != "Haley Spades Remy Rune Seductive Science Ember Moans Dezyred VR" {
		t.Errorf("query = %q", q)
	}

	// A shared folder is not.
	file.FolderIsScene = false
	if q := BuildQuery(file); strings.Contains(q, "Haley") {
		t.Errorf("a shared folder's name leaked into the query: %q", q)
	}

	long := BuildQuery(&FileInfo{Filename: strings.Repeat("word_", 80) + ".mp4"})
	if n := len(strings.Fields(long)); n > 46 {
		t.Errorf("query has %d words; Brave allows 50", n)
	}
	if strings.Count(long, "word") != 1 {
		t.Errorf("repeated words should appear once: %q", long)
	}
}
