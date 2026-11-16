package llmscrape

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestLive runs the fetch and extraction against a real page and a real model. It is skipped
// unless configured:
//
//	LLM_BASE_URL=http://host:8000/v1 LLM_MODEL=name LIVE_URL=https://... \
//	  [LIVE_FILENAME=file.mp4] [LIVE_DURATION_MIN=42] [LLM_API_KEY=...] go test -run TestLive -v ./pkg/llmscrape/
func TestLive(t *testing.T) {
	base, model, pageURL := os.Getenv("LLM_BASE_URL"), os.Getenv("LLM_MODEL"), os.Getenv("LIVE_URL")
	if base == "" || model == "" || pageURL == "" {
		t.Skip("set LLM_BASE_URL, LLM_MODEL and LIVE_URL to run")
	}
	client, err := NewClient(LLMConfig{BaseURL: base, Model: model, APIKey: os.Getenv("LLM_API_KEY"),
		DisableThinking: true, Timeout: 3 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	start := time.Now()
	page, err := NewFetcher("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		false, 30*time.Second).Fetch(ctx, pageURL, 12000)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("fetched in %v: %d images, %d videos, %d chars of text", time.Since(start), len(page.Images), len(page.Videos), len(page.Text))

	var file *FileInfo
	if fn := os.Getenv("LIVE_FILENAME"); fn != "" {
		d, _ := strconv.Atoi(os.Getenv("LIVE_DURATION_MIN"))
		file = &FileInfo{Filename: fn, DurationMinutes: d}
	}

	start = time.Now()
	raw, err := client.Complete(ctx, BuildMessages(page, file), "xbvr_scene", ExtractionSchema(len(page.Images), len(page.Videos), len(page.Links)), 4000)
	if err != nil {
		t.Fatal(err)
	}
	ext, err := ParseExtraction(raw, page)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("extracted in %v", time.Since(start))

	resolved := map[string]any{}
	if ext.CoverImage >= 0 {
		resolved["cover"] = page.Images[ext.CoverImage].URL
	}
	var gallery []string
	for _, i := range ext.GalleryImages {
		gallery = append(gallery, page.Images[i].URL)
	}
	resolved["gallery"] = gallery
	if ext.Trailer >= 0 {
		resolved["trailer"] = page.Videos[ext.Trailer].URL
	}
	out, _ := json.MarshalIndent(map[string]any{"extraction": ext, "resolved": resolved}, "", "  ")
	t.Logf("\n%s", out)
}
