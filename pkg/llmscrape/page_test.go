package llmscrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const fixturePage = `<!doctype html><html><head>
<title>Sweet Pink Bunny - Cleo Vega | ExampleVR</title>
<base href="https://cdn.example.com/scenes/">
<meta property="og:title" content="Sweet Pink Bunny">
<meta property="og:image" content="https://cdn.example.com/covers/bunny-cover.jpg">
<meta name="description" content="Cleo Vega in a 40 minute 8K scene.">
<link rel="canonical" href="https://examplevr.com/scene/1234-sweet-pink-bunny">
<script type="application/ld+json">
{"@type":"VideoObject","name":"Sweet Pink Bunny","thumbnailUrl":["https://cdn.example.com/thumbs/t1.jpg"],
 "contentUrl":"https://cdn.example.com/trailers/bunny.mp4","duration":"PT40M"}
</script>
<script>var tracking = "ignore me";</script>
</head><body>
<header><img src="/static/site-logo.png" alt="ExampleVR"></header>
<h1>Sweet Pink Bunny</h1>
<p>Starring   Cleo Vega.</p>
<img data-src="stills/s1.jpg" src="data:image/gif;base64,AAAA" alt="Still 1">
<img srcset="stills/s2-small.jpg 400w, stills/s2-large.jpg 1600w" alt="Still 2">
<div style="background-image: url('https://cdn.example.com/stills/s3.webp')"></div>
<img src="https://cdn.example.com/covers/bunny-cover.jpg" alt="duplicate of og:image">
<img src="/icons/favicon.png">
<video poster="stills/poster.jpg"><source src="trailers/bunny-720.mp4"></video>
<style>.x{color:red}</style>
</body></html>`

func TestExtractPage(t *testing.T) {
	u, _ := url.Parse("https://examplevr.com/scene/1234-sweet-pink-bunny?ref=search")
	p, err := ExtractPage(u, fixturePage, 5000)
	if err != nil {
		t.Fatal(err)
	}

	if p.Title != "Sweet Pink Bunny - Cleo Vega | ExampleVR" {
		t.Errorf("title = %q", p.Title)
	}
	if p.CanonicalURL != "https://examplevr.com/scene/1234-sweet-pink-bunny" {
		t.Errorf("canonical = %q", p.CanonicalURL)
	}
	if p.Meta["og:title"] != "Sweet Pink Bunny" || p.Meta["description"] == "" {
		t.Errorf("meta = %v", p.Meta)
	}
	if !strings.Contains(p.StructuredData, "VideoObject") {
		t.Errorf("structured data missing: %q", p.StructuredData)
	}

	var imgs []string
	for _, c := range p.Images {
		imgs = append(imgs, c.URL)
	}
	if len(imgs) == 0 || imgs[0] != "https://cdn.example.com/covers/bunny-cover.jpg" {
		t.Fatalf("og:image should be the first candidate, got %v", imgs)
	}
	want := []string{
		"https://cdn.example.com/thumbs/t1.jpg",              // JSON-LD thumbnailUrl
		"https://cdn.example.com/scenes/stills/s1.jpg",       // data-src resolved against <base>
		"https://cdn.example.com/scenes/stills/s2-large.jpg", // widest srcset entry
		"https://cdn.example.com/stills/s3.webp",             // inline background-image
		"https://cdn.example.com/scenes/stills/poster.jpg",   // video poster
	}
	for _, w := range want {
		if !containsStr(imgs, w) {
			t.Errorf("missing image candidate %s in %v", w, imgs)
		}
	}
	for _, bad := range []string{"site-logo", "favicon", "data:image", "s2-small"} {
		for _, u := range imgs {
			if strings.Contains(u, bad) {
				t.Errorf("candidate %s should have been filtered", u)
			}
		}
	}
	if countStr(imgs, "https://cdn.example.com/covers/bunny-cover.jpg") != 1 {
		t.Errorf("duplicate image candidates: %v", imgs)
	}

	var vids []string
	for _, c := range p.Videos {
		vids = append(vids, c.URL)
	}
	for _, w := range []string{"https://cdn.example.com/trailers/bunny.mp4", "https://cdn.example.com/scenes/trailers/bunny-720.mp4"} {
		if !containsStr(vids, w) {
			t.Errorf("missing video candidate %s in %v", w, vids)
		}
	}

	if strings.Contains(p.Text, "tracking") || strings.Contains(p.Text, "color:red") {
		t.Errorf("script/style leaked into text: %q", p.Text)
	}
	if !strings.Contains(p.Text, "Starring Cleo Vega.") {
		t.Errorf("text whitespace not collapsed: %q", p.Text)
	}
}

func TestExtractPageTruncatesText(t *testing.T) {
	u, _ := url.Parse("https://example.com/")
	p, err := ExtractPage(u, "<html><body>"+strings.Repeat("é ", 5000)+"</body></html>", 100)
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(p.Text)); n > 101 {
		t.Errorf("text is %d runes, want at most 101", n)
	}
}

func TestFetcherRefusesPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head><title>internal</title></head></html>"))
	}))
	defer srv.Close()

	guarded := NewFetcher("test", false, 5*time.Second)
	if _, err := guarded.Fetch(context.Background(), srv.URL, 1000); err == nil ||
		!strings.Contains(err.Error(), "private address") {
		t.Errorf("expected loopback to be refused, got %v", err)
	}

	open := NewFetcher("test", true, 5*time.Second)
	p, err := open.Fetch(context.Background(), srv.URL, 1000)
	if err != nil {
		t.Fatalf("allowPrivate fetch failed: %v", err)
	}
	if p.Title != "internal" {
		t.Errorf("title = %q", p.Title)
	}
}

func TestFetcherRejectsNonHTTP(t *testing.T) {
	f := NewFetcher("test", false, time.Second)
	for _, u := range []string{"file:///etc/passwd", "ftp://example.com/x", "gopher://x"} {
		if _, err := f.Fetch(context.Background(), u, 100); err == nil {
			t.Errorf("%s should be refused", u)
		}
	}
}

func containsStr(list []string, s string) bool { return countStr(list, s) > 0 }

func countStr(list []string, s string) int {
	n := 0
	for _, v := range list {
		if v == s {
			n++
		}
	}
	return n
}

func TestIsExpiringURL(t *testing.T) {
	cases := map[string]bool{
		"https://cdns.vrporn.com/videos/x/free_4k.mp4?ttl=1790701220&token=e107ed8c":  true,
		"https://bucket.s3.amazonaws.com/a.jpg?X-Amz-Expires=3600&X-Amz-Signature=ab": true,
		"https://cdn.example.com/a.jpg?Expires=1&Signature=x&Key-Pair-Id=k":           true,
		"https://cdn.example.com/cover.jpg":                                           false,
		"https://cdn.example.com/cover.jpg?v=3&w=1600":                                false,
	}
	for u, want := range cases {
		if got := IsExpiringURL(u); got != want {
			t.Errorf("IsExpiringURL(%s) = %v, want %v", u, got, want)
		}
	}
}

func TestExtractPageLinks(t *testing.T) {
	u, _ := url.Parse("https://example.com/studio/overview")
	html := `<html><body>
		<header><a href="/login">Sign in</a></header>
		<nav><a href="/scenes">All scenes</a></nav>
		<main>
			<a href="/scene/one"><img alt="Scene One"></a>
			<a href="https://other.example/scene/two">Scene Two</a>
			<a href="/trailers/preview.mp4">Trailer</a>
			<a href="/logo.png">Logo</a>
		</main>
	</body></html>`
	p, err := ExtractPage(u, html, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Links {
		got = append(got, c.URL)
	}
	want := []string{"https://example.com/scene/one", "https://other.example/scene/two"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("links = %v, want %v", got, want)
	}
	if len(p.Links) == 2 {
		if p.Links[0].Hint != "Scene One" {
			t.Errorf("image-only link hint = %q, want the image alt text", p.Links[0].Hint)
		}
		if p.Links[1].Hint != "Scene Two" {
			t.Errorf("link hint = %q", p.Links[1].Hint)
		}
	}
	if len(p.Videos) != 1 || p.Videos[0].URL != "https://example.com/trailers/preview.mp4" {
		t.Errorf("the trailer link belongs in the video candidates, got %+v", p.Videos)
	}
}

func TestExtractPageDropsExpiringCandidates(t *testing.T) {
	u, _ := url.Parse("https://example.com/scene")
	html := `<html><head><meta property="og:image" content="https://c/cover.jpg?token=abc&expires=9">
		<meta property="og:video" content="https://c/t.mp4?ttl=1&token=2"></head>
		<body><img src="https://c/still.jpg"><video src="https://c/durable.mp4"></video></body></html>`
	p, err := ExtractPage(u, html, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Images) != 1 || p.Images[0].URL != "https://c/still.jpg" {
		t.Errorf("images = %+v; the signed cover should be dropped", p.Images)
	}
	if len(p.Videos) != 1 || p.Videos[0].URL != "https://c/durable.mp4" {
		t.Errorf("videos = %+v; the signed trailer should be dropped", p.Videos)
	}
}
