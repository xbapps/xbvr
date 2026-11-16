package llmscrape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SearchResult is one web search hit.
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Searcher finds candidate scene pages.
type Searcher interface {
	Search(ctx context.Context, query string, count int) ([]SearchResult, error)
}

// BraveSearch uses the Brave Search API. Requests are spaced to respect the API's per-second
// limit and retried when it answers 429.
type BraveSearch struct {
	apiKey   string
	endpoint string
	http     *http.Client
	minGap   time.Duration
}

// NewBraveSearch returns a Brave Search client. The free plan allows one request per second.
func NewBraveSearch(apiKey string) *BraveSearch {
	return &BraveSearch{
		apiKey:   apiKey,
		endpoint: "https://api.search.brave.com/res/v1/web/search",
		http:     &http.Client{Timeout: 30 * time.Second},
		minGap:   1100 * time.Millisecond,
	}
}

// All Brave clients in the process share one rate limiter, since the limit is per API key and a
// batch run and the match screen can search at the same time.
var (
	braveMu   sync.Mutex
	braveLast time.Time
)

func (b *BraveSearch) wait(ctx context.Context) error {
	braveMu.Lock()
	defer braveMu.Unlock()
	if d := b.minGap - time.Since(braveLast); d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	braveLast = time.Now()
	return nil
}

type braveResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// Search runs a web search. Safe search is off: the pages being looked for are adult content.
func (b *BraveSearch) Search(ctx context.Context, query string, count int) ([]SearchResult, error) {
	if b.apiKey == "" {
		return nil, errors.New("Brave Search API key is not configured")
	}
	if count <= 0 || count > 20 {
		count = 10
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("count", strconv.Itoa(count))
	q.Set("safesearch", "off")

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if err := b.wait(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.endpoint+"?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Subscription-Token", b.apiKey)

		resp, err := b.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = errors.New("Brave Search rate limit")
			backoff := time.Duration(attempt+1) * 2 * time.Second
			if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s > 0 {
				backoff = time.Duration(s) * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Brave Search returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		}

		var br braveResponse
		if err := json.Unmarshal(body, &br); err != nil {
			return nil, fmt.Errorf("unexpected Brave Search response: %w", err)
		}
		out := make([]SearchResult, 0, len(br.Web.Results))
		for _, r := range br.Web.Results {
			out = append(out, SearchResult{
				Title:       collapse(htmlTagRe.ReplaceAllString(r.Title, "")),
				URL:         r.URL,
				Description: collapse(htmlTagRe.ReplaceAllString(r.Description, "")),
			})
		}
		return out, nil
	}
	return nil, fmt.Errorf("Brave Search failed after retries: %w", lastErr)
}

// commonFilenameWords are release and format tokens that say nothing about which scene a file is.
// This matches the list the match screen strips before searching the local index.
var commonFilenameWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`180 180x180 2880x1440 3d 3dh 3dv 30fps 30m 360 3840x1920 4k 5k 5400x2700
		60fps 6k 7k 7680x3840 8k fb360 fisheye190 funscript cmscript h264 h265 hevc hq hsp lq lr mkv mkx200
		mkx220 mono mp4 oculus oculus5k oculusrift original rf52 smartphone srt ssa tb uhd vrca220 vp9
		sbs fisheye 6k60 8k60 4k60 x265 x264 webrip vr180 vr360`) {
		commonFilenameWords[w] = true
	}
}

var (
	filenameSepRe   = regexp.MustCompile(`[._+,'’` + "`" + `\-\[\]()]+`)
	resolutionTokRe = regexp.MustCompile(`(?i)^\d{3,5}p(\d{2})?$`)
	extRe           = regexp.MustCompile(`(?i)(\.(mp4|mkv|avi|mov|wmv|m4v|webm))+$`)
	personNameRe    = regexp.MustCompile(`^([A-Z][a-z]{2,})([A-Z][a-z]{2,})$`)
	longNumberRe    = regexp.MustCompile(`\d{5,}`)
)

// CleanTerms turns a video filename or folder name into search terms. It drops the extension,
// separators, format and resolution tokens, and release-name metadata: everything after a
// standalone XXX, and tokens carrying five or more digits, which are IDs and hashes rather than
// words. Performer names run together as FirstLast are split.
func CleanTerms(name string) string {
	name = extRe.ReplaceAllString(name, "")
	var out []string
	for _, w := range strings.Fields(filenameSepRe.ReplaceAllString(name, " ")) {
		lw := strings.ToLower(w)
		if lw == "xxx" {
			break
		}
		if commonFilenameWords[lw] || resolutionTokRe.MatchString(lw) || longNumberRe.MatchString(w) {
			continue
		}
		if m := personNameRe.FindStringSubmatch(w); m != nil {
			out = append(out, m[1], m[2])
			continue
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}

// FilenameTerms is CleanTerms for a filename.
func FilenameTerms(filename string) string { return CleanTerms(filename) }

// BuildQuery builds the web search query for a file from the user's saved context, the folder
// name when the folder is the scene's own, and the filename, in that order of trust, without
// repeating a word, within Brave's 50-word limit.
func BuildQuery(file *FileInfo) string {
	words := dedupeWords(strings.Fields(strings.Join([]string{
		strings.TrimSpace(file.Context), file.FolderTerms(), FilenameTerms(file.Filename),
	}, " ")))
	hasVR := false
	for _, w := range words {
		if strings.EqualFold(w, "vr") {
			hasVR = true
		}
	}
	if len(words) > 45 {
		words = words[:45]
	}
	if !hasVR {
		words = append(words, "VR")
	}
	return truncate(strings.Join(words, " "), 390)
}

// LocalSearchTerms is the folder and filename part of BuildQuery, for XBVR's own scene search.
func LocalSearchTerms(file *FileInfo) string {
	return strings.Join(dedupeWords(strings.Fields(file.FolderTerms()+" "+FilenameTerms(file.Filename))), " ")
}

func dedupeWords(words []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range words {
		k := strings.ToLower(w)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, w)
	}
	return out
}
